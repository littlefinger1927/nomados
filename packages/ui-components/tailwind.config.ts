import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        nomados: {
          primary: '#3b82f6',
          danger: '#ef4444',
          surface: '#1e293b',
          background: '#0a0e17',
          border: '#334155',
          text: '#e2e8f0',
          'text-muted': '#94a3b8',
        },
      },
    },
  },
  plugins: [],
};

export default config;