import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'NomadOS',
  description: 'Sovereign Compute Platform',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-nomados-background font-sans text-nomados-text antialiased">
        {children}
      </body>
    </html>
  );
}