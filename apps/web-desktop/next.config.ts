import path from 'path';
import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // Transpile the ui-components package since it uses JSX/TSX directly
  transpilePackages: ['@nomados/ui-components'],
  // Set workspace root to suppress lockfile inference warning
  outputFileTracingRoot: path.join(__dirname, '../../'),
  // Enable standalone output for smaller Docker images
  output: 'standalone',
};

export default nextConfig;