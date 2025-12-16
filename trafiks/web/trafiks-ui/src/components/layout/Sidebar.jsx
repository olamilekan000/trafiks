import { NavLink } from "react-router-dom";
import "./Sidebar.css";

export default function Sidebar({ items = [] }) {
  return (
    <aside className="sidebar">
      <nav className="sidebar-nav">
        {items.map((item) => {
          const IconComponent = item.icon;
          return (
            <NavLink
              key={item.path}
              to={item.path}
              className={({ isActive }) =>
                `sidebar-item ${isActive ? "sidebar-item-active" : ""}`
              }
            >
              {IconComponent && (
                <span className="sidebar-icon" aria-label={item.label}>
                  <IconComponent size={20} />
                </span>
              )}
              <span>{item.label}</span>
            </NavLink>
          );
        })}
      </nav>
    </aside>
  );
}
