import React from 'react';

interface CardProps {
  title?: string;
  children: React.ReactNode;
  className?: string;
  actions?: React.ReactNode;
}

export const Card: React.FC<CardProps> = ({
  title,
  children,
  className = '',
  actions,
}) => {
  return (
    <div
      className={`
        rounded-lg border border-nomados-border bg-nomados-surface
        ${className}
      `.trim()}
    >
      {(title || actions) && (
        <div className="flex items-center justify-between border-b border-nomados-border px-6 py-4">
          {title && (
            <h3 className="text-lg font-semibold text-nomados-text">
              {title}
            </h3>
          )}
          {actions && (
            <div className="flex items-center gap-2">
              {actions}
            </div>
          )}
        </div>
      )}
      <div className="px-6 py-4">
        {children}
      </div>
    </div>
  );
};

export default Card;