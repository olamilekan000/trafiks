import "./IconContainer.css";

export default function IconContainer({
  children,
  size = "medium",
  className = "",
}) {
  return (
    <div className={`icon-container icon-container-${size} ${className}`}>
      {children}
    </div>
  );
}
