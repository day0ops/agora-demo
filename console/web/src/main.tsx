import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { IdentityProvider } from "./hooks/useIdentity";
import { Landing } from "./routes/Landing";
import { Shop } from "./routes/Shop";
import { Sell } from "./routes/Sell";
import { Ops } from "./routes/Ops";
import "./style.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <IdentityProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/shop" element={<Shop />} />
          <Route path="/sell" element={<Sell />} />
          <Route path="/ops" element={<Ops />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </IdentityProvider>
  </StrictMode>,
);
