import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // No static export — the app runs against a Go backend API at runtime
  eslint: {
    ignoreDuringBuilds: true,
  },
  typescript: {
    ignoreBuildErrors: true,
  }
};

export default nextConfig;
