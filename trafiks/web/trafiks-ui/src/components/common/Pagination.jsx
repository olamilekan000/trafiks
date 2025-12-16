import Button from "./Button";
import "./Pagination.css";

export default function Pagination({
  currentPage,
  totalPages,
  total,
  onPageChange,
  limit,
  className = "",
}) {
  if (totalPages <= 1) return null;

  return (
    <div className={`pagination ${className}`}>
      <Button
        variant="ghost"
        size="small"
        onClick={() => onPageChange(currentPage - 1)}
        disabled={currentPage === 1}
      >
        Previous
      </Button>
      <span className="pagination-info">
        Page {currentPage} of {totalPages} ({total} total)
      </span>
      <Button
        variant="ghost"
        size="small"
        onClick={() => onPageChange(currentPage + 1)}
        disabled={currentPage >= totalPages}
      >
        Next
      </Button>
    </div>
  );
}
