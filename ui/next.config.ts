import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  typescript: {
    ignoreBuildErrors: true,
  },
  logging: {
    fetches: {
      fullUrl: true,
    },
  },
  experimental: { swcPlugins: [] },
  turbopack: undefined,
  reactCompiler: true,
  compiler: { removeConsole: process.env.NODE_ENV === "production" },
};

export default nextConfig;
