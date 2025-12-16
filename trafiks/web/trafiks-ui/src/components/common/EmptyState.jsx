import Button from "./Button";
import "./EmptyState.css";

export default function EmptyState({
  icon: Icon,
  title,
  description,
  actionLabel,
  onAction,
  className = "",
}) {
  return (
    <div className={`empty-state ${className}`}>
      {Icon && (
        <div className="empty-state-icon">
          <Icon size={48} style={{ color: "var(--text-secondary)" }} />
        </div>
      )}
      {title && <p className="empty-state-title">{title}</p>}
      {description && <p className="empty-state-description">{description}</p>}
      {actionLabel && onAction && (
        <Button
          variant="primary"
          onClick={onAction}
          className="empty-state-action"
        >
          {actionLabel}
        </Button>
      )}
    </div>
  );
}
