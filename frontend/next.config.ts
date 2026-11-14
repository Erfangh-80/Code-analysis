import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // The browser always uses relative /api URLs. Next proxies them to the
  // backend on the same machine, keeping repository paths off remote services.
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: "http://127.0.0.1:8080/api/:path*",
      },
    ];
  },
  allowedDevOrigins: ["*.e2b.app"],
};

export default nextConfig;
