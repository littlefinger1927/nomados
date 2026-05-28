'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { Button } from '@nomados/ui-components';
import { useToast } from '@nomados/ui-components';
import { apiFetch, getAccessToken } from '../../../lib/auth';
import { encrypt, decrypt } from '../../../lib/crypto';
import { getWorkspaceKey, getCachedMasterKey, initializeMasterKey } from '../../../lib/keys';

interface FileInfo {
  id: { value: string };
  filename: string;
  size: number;
  created_at: number;
  updated_at: number;
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

function formatDate(epoch: number): string {
  return new Date(epoch * 1000).toLocaleDateString('en-US', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
  });
}

interface FileBrowserProps {
  workspaceId: string;
  workspaceState: string;
}

export function FileBrowser({ workspaceId, workspaceState }: FileBrowserProps) {
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [encryptionAvailable, setEncryptionAvailable] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { addToast } = useToast();

  // Initialize master key from session and check encryption availability
  useEffect(() => {
    initializeMasterKey().then((available) => {
      setEncryptionAvailable(available);
    });
  }, []);

  const fetchFiles = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch('/v1/file/list', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ workspace_id: { value: workspaceId } }),
      });
      const data = await res.json();
      setFiles(data.files || []);
    } catch {
      setFiles([]);
    } finally {
      setLoading(false);
    }
  }, [workspaceId]);

  useEffect(() => {
    if (workspaceState === 'Running') {
      fetchFiles();
    }
  }, [workspaceState, fetchFiles]);

  if (workspaceState !== 'Running') {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <svg className="h-10 w-10 text-nomados-text-muted mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        <p className="text-sm text-nomados-text-muted">Start this workspace to browse files</p>
      </div>
    );
  }

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    try {
      // Read file as ArrayBuffer for encryption
      const fileData = await file.arrayBuffer();

      // Derive workspace key for encryption
      const workspaceKey = await getWorkspaceKey(workspaceId);

      let payload: Blob;
      if (workspaceKey) {
        // Encrypt file with workspace key
        const encrypted = await encrypt(fileData, workspaceKey);
        payload = new Blob([encrypted]);
      } else {
        // No master key available — upload unencrypted (with warning)
        payload = file;
      }

      const token = getAccessToken();
      const formData = new FormData();
      formData.append('workspace_id', workspaceId);
      formData.append('file', payload, file.name);
      const res = await apiFetch('/v1/file/upload', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
        body: formData,
      });
      if (!res.ok) throw new Error('Upload failed');
      const encryptedLabel = workspaceKey ? ' (encrypted)' : '';
      addToast(`Uploaded ${file.name}${encryptedLabel}`, 'success');
      fetchFiles();
    } catch {
      addToast('Failed to upload file', 'error');
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  const handleDownload = async (fileId: string, filename: string) => {
    try {
      // Fetch file (may be encrypted)
      const res = await apiFetch(
        `/v1/file/download/?workspace_id=${workspaceId}&file_id=${fileId}`,
        {},
      );
      if (!res.ok) throw new Error('Download failed');
      const rawData = await res.arrayBuffer();

      // Derive workspace key for decryption
      const workspaceKey = await getWorkspaceKey(workspaceId);

      let fileData: ArrayBuffer;
      if (workspaceKey) {
        // Attempt decryption — if the file wasn't encrypted, decrypt will throw
        try {
          fileData = await decrypt(rawData, workspaceKey);
        } catch {
          // Decryption failed — file may have been uploaded before encryption
          // was enabled. Serve the raw data as-is.
          fileData = rawData;
        }
      } else {
        // No master key available — serve raw data
        fileData = rawData;
      }

      // Trigger browser download
      const blob = new Blob([fileData]);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      addToast('Failed to download file', 'error');
    }
  };

  const handleDelete = async (fileId: string, filename: string) => {
    if (!confirm(`Delete "${filename}"? This cannot be undone.`)) return;
    try {
      await apiFetch('/v1/file/delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ workspace_id: { value: workspaceId }, file_id: { value: fileId } }),
      });
      addToast(`Deleted ${filename}`, 'success');
      fetchFiles();
    } catch {
      addToast('Failed to delete file', 'error');
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <span className="text-xs font-medium text-nomados-text-muted uppercase tracking-wider">Files</span>
          {encryptionAvailable ? (
            <svg className="h-3.5 w-3.5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-label="End-to-end encryption active">
              <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
            </svg>
          ) : (
            <svg className="h-3.5 w-3.5 text-yellow-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2} aria-label="Encryption key not available">
              <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 10.5V6.75a4.5 4.5 0 119 0v3.75M3.75 21.75h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H3.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
            </svg>
          )}
        </div>
        <div>
          <input
            ref={fileInputRef}
            type="file"
            className="hidden"
            onChange={handleUpload}
            disabled={uploading}
          />
          <Button variant="secondary" size="sm" onClick={() => fileInputRef.current?.click()} loading={uploading}>
            Upload
          </Button>
        </div>
      </div>

      {!encryptionAvailable && (
        <p className="text-xs text-yellow-500/80 leading-relaxed">
          Encryption key not available — files will be uploaded unencrypted.
        </p>
      )}

      {loading ? (
        <div className="flex justify-center py-4">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-nomados-border border-t-nomados-primary" />
        </div>
      ) : files.length === 0 ? (
        <div className="flex flex-col items-center py-6 text-center">
          <svg className="h-10 w-10 text-nomados-text-muted mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
          <p className="text-sm text-nomados-text-muted">No files yet</p>
          <p className="text-xs text-nomados-text-muted mt-1">Upload a file to get started</p>
        </div>
      ) : (
        <div className="flex flex-col gap-1">
          {files.map((file) => (
            <div
              key={file.id.value}
              className="flex items-center gap-2 rounded-lg px-3 py-2 hover:bg-nomados-border/30 group"
            >
              <svg className="h-4 w-4 shrink-0 text-nomados-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
              </svg>
              <div className="flex-1 min-w-0">
                <p className="text-sm text-nomados-text truncate">{file.filename}</p>
                <p className="text-xs text-nomados-text-muted">{formatSize(file.size)} · {formatDate(file.created_at)}</p>
              </div>
              <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  onClick={() => handleDownload(file.id.value, file.filename)}
                  className="p-1 rounded hover:bg-nomados-border/50 text-nomados-text-muted hover:text-nomados-text"
                  title="Download"
                >
                  <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
                  </svg>
                </button>
                <button
                  onClick={() => handleDelete(file.id.value, file.filename)}
                  className="p-1 rounded hover:bg-nomados-border/50 text-nomados-text-muted hover:text-red-400"
                  title="Delete"
                >
                  <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                  </svg>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}