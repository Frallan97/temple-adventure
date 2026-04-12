interface StarRatingProps {
  rating: number;
  maxStars?: number;
  size?: "sm" | "md" | "lg";
  interactive?: boolean;
  onRate?: (rating: number) => void;
}

const sizeClasses = {
  sm: "text-sm",
  md: "text-xl",
  lg: "text-2xl",
};

export function StarRating({
  rating,
  maxStars = 5,
  size = "md",
  interactive = false,
  onRate,
}: StarRatingProps) {
  return (
    <div className={`flex gap-0.5 ${sizeClasses[size]} ${interactive ? "cursor-pointer" : ""}`}>
      {Array.from({ length: maxStars }, (_, i) => {
        const starIndex = i + 1;
        const filled = starIndex <= Math.round(rating);
        return (
          <span
            key={i}
            onClick={interactive && onRate ? () => onRate(starIndex) : undefined}
            className={`${
              filled ? "text-amber-400" : "text-gray-600"
            } ${interactive ? "hover:text-amber-300 transition-colors" : ""}`}
          >
            {filled ? "\u2605" : "\u2606"}
          </span>
        );
      })}
    </div>
  );
}
