import { Link } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
import { IconBrand, IconSun, IconMoon, IconChevronDown } from "../icons";
import { useTheme } from "../hooks/useTheme";
import { useIdentity, canSell, canOperate } from "../hooks/useIdentity";

export const APP_NAME = "Agora";

type View = "index" | "shop" | "sell" | "ops";

const NAV: { to: string; key: View; label: string }[] = [
  { to: "/shop", key: "shop", label: "Shop" },
  { to: "/sell", key: "sell", label: "Sell" },
  { to: "/ops", key: "ops", label: "Operator" },
];

export function Navbar({ active }: { active: View }) {
  const { theme, toggleTheme } = useTheme();
  const { identity } = useIdentity();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    function onDocClick(e: MouseEvent) {
      if (!menuRef.current?.contains(e.target as Node)) setMenuOpen(false);
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") setMenuOpen(false);
    }
    document.addEventListener("click", onDocClick);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("click", onDocClick);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [menuOpen]);

  function navVisible(key: View): boolean {
    if (key === "sell") return canSell(identity);
    if (key === "ops") return canOperate(identity);
    return true;
  }

  return (
    <div className="topbar">
      <Link className="brand" to="/">
        <IconBrand />
        {APP_NAME}
      </Link>
      <nav>
        {NAV.map((n) => (
          <Link
            key={n.key}
            to={n.to}
            data-key={n.key}
            className={n.key === active ? "active" : ""}
            style={{ display: navVisible(n.key) ? "" : "none" }}
          >
            {n.label}
          </Link>
        ))}
      </nav>
      <div className="account-area">
        {identity ? (
          <div className="account-menu" ref={menuRef}>
            <button
              type="button"
              className="ghost account-menu-trigger"
              aria-haspopup="true"
              aria-expanded={menuOpen}
              onClick={(e) => {
                e.stopPropagation();
                setMenuOpen((open) => !open);
              }}
            >
              <span className="account-avatar">
                {identity.username.charAt(0).toUpperCase()}
              </span>
              <span className="account-name">{identity.username}</span>
              <IconChevronDown />
            </button>
            {menuOpen && (
              <div className="account-menu-panel">
                <a className="account-menu-item" href="/account/logout">
                  Log out
                </a>
              </div>
            )}
          </div>
        ) : (
          <a className="ghost account-link" href="/account/login">
            Login
          </a>
        )}
      </div>
      <button
        className="ghost"
        type="button"
        aria-label={
          theme === "dark" ? "Switch to light theme" : "Switch to dark theme"
        }
        onClick={toggleTheme}
      >
        {theme === "dark" ? <IconMoon /> : <IconSun />}
      </button>
    </div>
  );
}
