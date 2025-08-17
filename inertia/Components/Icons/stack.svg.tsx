import React from 'react';

export const StackIcon = React.forwardRef<
  React.ElementRef<'svg'>,
  React.ComponentPropsWithoutRef<'svg'>
>((props, forwardedRef) => {
  return (
    <svg
      width="24px"
      height="24px"
      viewBox="0 0 64 64"
      xmlns="http://www.w3.org/2000/svg"
      strokeWidth="3"
      stroke="currentColor"
      fill="none"
      {...props}
      ref={forwardedRef}
    >
      <title>Stack Icon</title>
      <path
        d="M11.39,19.82,32.1,30.48a.51.51,0,0,0,.45,0L54.21,19.81a.5.5,0,0,0,0-.89L33.72,8.45a2,2,0,0,0-1.82,0L11.4,18.93A.5.5,0,0,0,11.39,19.82Z"
        strokeLinecap="round"
      />
      <path
        d="M10.83,32.23l21.27,11h.45l22.25-11"
        strokeLinecap="round"
      />
      <path
        d="M10.83,44.94l21.27,11h.45l22.25-11"
        strokeLinecap="round"
      />
    </svg>
  );
});
