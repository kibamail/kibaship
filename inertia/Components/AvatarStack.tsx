import { useState, Children, type ReactNode } from 'react';

interface AvatarStackProps {
  children: ReactNode;
  className?: string;
  avatarSize?: 'sm' | 'md' | 'lg';
  maxVisible?: number;
}

export default function AvatarStack({
  children,
  className = '',
  avatarSize = 'md',
  maxVisible,
}: AvatarStackProps) {
  const [isHovered, setIsHovered] = useState(false);

  const sizeClasses = {
    sm: 'w-4 h-4',
    md: 'w-5 h-5',
    lg: 'w-6 h-6',
  };

  const childrenArray = Children.toArray(children);
  const visibleChildren = maxVisible ? childrenArray.slice(0, maxVisible) : childrenArray;
  const remainingCount = maxVisible && childrenArray.length > maxVisible ? childrenArray.length - maxVisible : 0;

  const getStackedStyle = (index: number) => {
    if (isHovered) {
      return {
        transform: `translateX(${index * 8}px)`,
        zIndex: visibleChildren.length - index,
      };
    }

    let translateX = 0;
    if (index === 1) {
      // Image 2: 50% visible
      translateX = -12;
    } else if (index >= 2) {
      // Image 3 and above: 10% visible
      translateX = -16 - (index - 2) * 2;
    }

    return {
      transform: `translateX(${translateX}px)`,
      zIndex: visibleChildren.length - index,
    };
  };

  const getOverflowStyle = () => {
    if (isHovered) {
      return {
        transform: `translateX(${visibleChildren.length * 8}px)`,
        zIndex: 0,
      };
    }

    const lastVisibleIndex = visibleChildren.length - 1;
    let translateX = 0;
    if (lastVisibleIndex === 1) {
      translateX = -12;
    } else if (lastVisibleIndex >= 2) {
      translateX = -16 - (lastVisibleIndex - 2) * 2;
    }

    return {
      transform: `translateX(${translateX - 2}px)`,
      zIndex: 0,
    };
  };

  return (
    <div
      className={`flex items-center ${className}`}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <div className="relative flex items-center">
        {visibleChildren.map((child, index) => (
          <div
            key={index}
            className={`relative transition-transform duration-300 ease-in-out`}
            style={getStackedStyle(index)}
          >
            <div className="w-full h-full rounded-full overflow-hidden">
              {child}
            </div>
          </div>
        ))}
        
        {remainingCount > 0 && (
          <div
            className={`relative transition-transform duration-300 ease-in-out`}
            style={getOverflowStyle()}
          >
            <div className="w-full h-full rounded-full flex items-center justify-center">
              <span className="text-xs font-medium">
                +{remainingCount}
              </span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
