import React from 'react';

export const LeaseWebIcon = React.forwardRef<
  React.ElementRef<'svg'>,
  React.ComponentPropsWithoutRef<'svg'>
>((props, forwardedRef) => {
  return (
    <svg
      width="32px"
      height="32px"
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      {...props}
      ref={forwardedRef}
    >
      <title>LeaseWeb logo</title>
      <rect width="24" height="24" rx="4" fill="#0066CC" />
      <path d="M6 8h2v6h4v2H6V8zm6 0h2v8h-2V8zm4 0h2v3h-2V8zm0 5h2v3h-2v-3z" fill="white" />
    </svg>
  );
});
