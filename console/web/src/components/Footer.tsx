import { useEffect, useState } from "react";
import { getVersion } from "../api";
import { APP_NAME } from "./Navbar";

export function Footer() {
  const [version, setVersion] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    getVersion()
      .then((v) => {
        if (!cancelled) setVersion(v.version);
      })
      .catch(() => {
        // best-effort; the footer just omits the version if unreachable
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <footer className="site-footer">
      <span>{APP_NAME}</span>
      <span>{version ?? ""}</span>
      <a
        href="https://github.com/day0ops/agora-demo"
        target="_blank"
        rel="noopener noreferrer"
      >
        GitHub
      </a>
    </footer>
  );
}
