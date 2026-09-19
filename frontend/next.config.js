/** @type {import('next').NextConfig} */
const nextConfig = {
  // No proxy needed - using direct API routes to Python agent
  eslint: {
    // Disable ESLint during builds for faster production builds
    ignoreDuringBuilds: true,
  },
  typescript: {
    // Disable type checking during builds for faster production builds
    ignoreBuildErrors: true,
  },
};

module.exports = nextConfig; 
