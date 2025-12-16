import { useEffect } from "react";
import { FiX } from "react-icons/fi";
import Button from "./Button";
import "./Modal.css";

export default function Modal({
  isOpen,
  onClose,
  title,
  children,
  footer,
  size = "medium",
  showCloseButton = true,
  className = "",
}) {
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div className={`modal-overlay ${className}`} onClick={onClose}>
      <div
        className={`modal-content modal-${size}`}
        onClick={(e) => e.stopPropagation()}
      >
        {title && (
          <div className="modal-header">
            <h2>{title}</h2>
            {showCloseButton && (
              <Button variant="ghost" size="small" onClick={onClose}>
                <FiX size={20} />
              </Button>
            )}
          </div>
        )}
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-footer">{footer}</div>}
      </div>
    </div>
  );
}
