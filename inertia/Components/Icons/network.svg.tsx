import React from 'react'

export const NetworkIcon = React.forwardRef<
  React.ElementRef<'svg'>,
  React.ComponentPropsWithoutRef<'svg'>
>((props, forwardedRef) => {
  return (
    <svg
      width="24px"
      height="24px"
      strokeWidth="1.5"
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      color="currentColor"
      {...props}
      ref={forwardedRef}
    >
      <rect
        x="3"
        y="2"
        width="7"
        height="5"
        rx="0.6"
        stroke="currentColor"
        strokeWidth="1.5"
      />
      <rect
        x="8.5"
        y="17"
        width="7"
        height="5"
        rx="0.6"
        stroke="currentColor"
        strokeWidth="1.5"
      />
      <rect
        x="14"
        y="2"
        width="7"
        height="5"
        rx="0.6"
        stroke="currentColor"
        strokeWidth="1.5"
      />
      <path
        d="M6.5 7V10.5C6.5 11.6046 7.39543 12.5 8.5 12.5H15.5C16.6046 12.5 17.5 11.6046 17.5 10.5V7"
        stroke="currentColor"
        strokeWidth="1.5"
      />
      <path
        d="M12 12.5V17"
        stroke="currentColor"
        strokeWidth="1.5"
      />
    </svg>
  )
})
