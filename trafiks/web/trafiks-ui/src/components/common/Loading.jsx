import "./Loading.css";
import infinitySvg from "../../assets/Infinity.svg";

export default function Loading({ className = "", size = "medium" }) {
  return (
    <div className={`loading-container ${className}`}>
      <img src={infinitySvg} alt="Loading..." className={`loading-${size}`} />
    </div>
  );
}
