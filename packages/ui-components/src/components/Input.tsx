import React from 'react';

interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type' | 'size'> {
  label?: string;
  type?: 'text' | 'password' | 'email';
  error?: string;
}

export const Input: React.FC<InputProps> = ({
  label,
  type = 'text',
  value,
  onChange,
  placeholder,
  error,
  disabled = false,
  className = '',
  id,
  ...rest
}) => {
  const inputId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined);

  return (
    <div className={`flex flex-col gap-1.5 ${className}`}>
      {label && (
        <label
          htmlFor={inputId}
          className="text-sm font-medium text-nomados-text"
        >
          {label}
        </label>
      )}
      <input
        id={inputId}
        type={type}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        disabled={disabled}
        className={`
          w-full rounded-lg border bg-nomados-surface px-3 py-2 text-sm
          text-nomados-text placeholder-nomados-text-muted
          transition-colors duration-150
          focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-nomados-background
          disabled:cursor-not-allowed disabled:opacity-50
          ${
            error
              ? 'border-nomados-danger focus:ring-nomados-danger/50'
              : 'border-nomados-border focus:ring-nomados-primary/50 focus:border-nomados-primary'
          }
        `.trim()}
        aria-invalid={error ? 'true' : undefined}
        aria-describedby={error && inputId ? `${inputId}-error` : undefined}
        {...rest}
      />
      {error && (
        <p
          id={inputId ? `${inputId}-error` : undefined}
          className="text-sm text-nomados-danger"
          role="alert"
        >
          {error}
        </p>
      )}
    </div>
  );
};

export default Input;