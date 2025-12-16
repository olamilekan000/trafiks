import "./Card.css";

export default function Card({
  children,
  className = "",
  onClick,
  interactive = false,
}) {
  const classes = `card ${interactive ? "card-interactive" : ""} ${className}`;

  return (
    <div className={classes} onClick={onClick}>
      {children}
    </div>
  );
}
