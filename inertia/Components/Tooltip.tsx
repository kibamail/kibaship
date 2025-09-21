'use client'

import * as React from 'react'
import * as TooltipPrimitive from '@radix-ui/react-tooltip'
import cn from 'classnames'

interface TooltipProps {
  children: React.ReactNode
  content: React.ReactNode
  side?: 'top' | 'right' | 'bottom' | 'left'
  align?: 'start' | 'center' | 'end'
  sideOffset?: number
  delayDuration?: number
  skipDelayDuration?: number
  open?: boolean
  defaultOpen?: boolean
  onOpenChange?: (open: boolean) => void
  className?: string
}

/**
 * Tooltip Component
 *
 * A flexible tooltip component built on top of @radix-ui/react-tooltip.
 * Provides a simple API for adding tooltips to any element.
 *
 * Features:
 * - Simple usage: wrap any element and provide content
 * - Customizable positioning (side, align, offset)
 * - Configurable timing (delay, skip delay)
 * - Controlled or uncontrolled state
 * - Consistent styling with Owly design system
 *
 * @example
 * <Tooltip content="This is a tooltip">
 *   <button>Hover me</button>
 * </Tooltip>
 *
 * @example
 * <Tooltip content={<div>Rich <strong>HTML</strong> content</div>} side="right">
 *   <span>Complex tooltip</span>
 * </Tooltip>
 */
export function Tooltip({
  children,
  content,
  side = 'top',
  align = 'center',
  sideOffset = 8,
  delayDuration = 400,
  skipDelayDuration = 300,
  open,
  defaultOpen,
  onOpenChange,
  className,
  ...props
}: TooltipProps) {
  return (
    <TooltipPrimitive.Provider delayDuration={delayDuration} skipDelayDuration={skipDelayDuration}>
      <TooltipPrimitive.Root open={open} defaultOpen={defaultOpen} onOpenChange={onOpenChange}>
        <TooltipPrimitive.Trigger asChild>{children}</TooltipPrimitive.Trigger>
        <TooltipPrimitive.Portal>
          <TooltipPrimitive.Content
            side={side}
            align={align}
            sideOffset={sideOffset}
            className={cn(
              // Base styles
              'z-50 overflow-hidden rounded-lg p-3 text-sm max-w-xs',
              // Colors using Owly design tokens
              'text-owly-content-primary-inverse bg-owly-background-inverse',
              // Animation
              'animate-in fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95',
              // Side-specific animations
              'data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
              className
            )}
            {...props}
          >
            {content}
            <TooltipPrimitive.Arrow
              className="fill-owly-background-inverse"
              width={11}
              height={5}
            />
          </TooltipPrimitive.Content>
        </TooltipPrimitive.Portal>
      </TooltipPrimitive.Root>
    </TooltipPrimitive.Provider>
  )
}

/**
 * TooltipProvider Component
 *
 * Use this when you need to wrap multiple tooltips with shared configuration.
 * This is useful when you have many tooltips in a component tree and want
 * to avoid creating multiple providers.
 *
 * @example
 * <TooltipProvider>
 *   <SimpleTooltip content="First tooltip">
 *     <button>Button 1</button>
 *   </SimpleTooltip>
 *   <SimpleTooltip content="Second tooltip">
 *     <button>Button 2</button>
 *   </SimpleTooltip>
 * </TooltipProvider>
 */
export function TooltipProvider({
  children,
  delayDuration = 400,
  skipDelayDuration = 300,
}: {
  children: React.ReactNode
  delayDuration?: number
  skipDelayDuration?: number
}) {
  return (
    <TooltipPrimitive.Provider delayDuration={delayDuration} skipDelayDuration={skipDelayDuration}>
      {children}
    </TooltipPrimitive.Provider>
  )
}
