import { Outlet } from "react-router-dom";
import Header from "./Header";
import "./MainLayout.css";

export default function MainLayout() {
  return (
    <div className="main-layout">
      <Header />
      <div className="main-content">
        <Outlet />
      </div>
    </div>
  );
}
