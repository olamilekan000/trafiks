import "./Table.css";

export default function Table({ children, className = "" }) {
  return (
    <div className="table-container">
      <table className={`table ${className}`}>{children}</table>
    </div>
  );
}
