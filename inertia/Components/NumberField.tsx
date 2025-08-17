import { MinusCircleIcon } from '~/Components/Icons/minus-circle.svg'
import { PlusCircleIcon } from '~/Components/Icons/plus-circle.svg'
import { Button } from '@kibamail/owly'
import * as TextField from '@kibamail/owly/text-field'
import React, { createContext, type PropsWithChildren, useContext, useState } from 'react'

/**
 * NumberField Component
 *
 * A compound component for numeric input with increment/decrement buttons.
 * Built on top of TextField from @kibamail/owly with additional numeric controls.
 *
 * Features:
 * - Direct typing support with validation
 * - Increment/decrement buttons with custom icons
 * - Min/max value constraints with automatic clamping
 * - Proper form integration with hidden input
 * - Full accessibility support through TextField base
 *
 * @example
 * <NumberField.Root name="count" value={5} min={1} max={10} onChange={setValue}>
 *   <NumberField.Label>Item Count</NumberField.Label>
 *   <NumberField.Field placeholder="Enter count">
 *     <NumberField.DecrementButton />
 *     <NumberField.IncrementButton />
 *     <NumberField.Hint>Choose between 1 and 10 items</NumberField.Hint>
 *     {error && <NumberField.Error>{error}</NumberField.Error>}
 *   </NumberField.Field>
 * </NumberField.Root>
 */

interface NumberFieldContextValue {
  value: number
  min: number
  max: number
  disabled: boolean
  name?: string
  onChange: (value: number) => void
  onInputChange: (e: React.ChangeEvent<HTMLInputElement>) => void
}

const NumberFieldContext = createContext<NumberFieldContextValue | null>(null)

const useNumberFieldContext = () => {
  const context = useContext(NumberFieldContext)
  if (!context) {
    throw new Error('NumberField components must be used within NumberField.Root')
  }
  return context
}

/**
 * Root component that provides context for all NumberField components.
 * Manages the numeric value state and provides validation logic.
 *
 * @param value - Current numeric value
 * @param min - Minimum allowed value (default: 0)
 * @param max - Maximum allowed value (default: 100)
 * @param disabled - Whether the field is disabled
 * @param name - Form field name for submission
 * @param onChange - Callback when value changes
 * @param children - Child components (Label, Field, etc.)
 */
interface NumberFieldRootProps extends PropsWithChildren {
  value: number
  min?: number
  max?: number
  name?: string
  disabled?: boolean
  onChange?: (value: number) => void
}

const NumberFieldRoot = React.forwardRef<HTMLDivElement, NumberFieldRootProps>(
  ({ value, min = 0, max = 100, disabled = false, name, onChange, children }, ref) => {
    const [currentValue, setCurrentValue] = useState(value)

    function onValueChange(newValue: number) {
      const clampedValue = Math.min(Math.max(newValue, min), max)
      setCurrentValue(clampedValue)
      onChange?.(clampedValue)
    }

    function onInputChange(e: React.ChangeEvent<HTMLInputElement>) {
      const inputValue = e.target.value
      if (inputValue === '') {
        setCurrentValue(min)
        onChange?.(min)
        return
      }

      const numValue = Number.parseInt(inputValue, 10)
      if (!Number.isNaN(numValue)) {
        onValueChange(numValue)
      }
    }

    const contextValue: NumberFieldContextValue = {
      value: currentValue,
      min,
      max,
      disabled,
      name,
      onChange: onValueChange,
      onInputChange: onInputChange,
    }

    return (
      <NumberFieldContext.Provider value={contextValue}>
        <div ref={ref}>{children}</div>
      </NumberFieldContext.Provider>
    )
  }
)

NumberFieldRoot.displayName = 'NumberField.Root'

/**
 * Label component for the NumberField.
 * Uses TextField.Label from owly for consistent styling.
 */
const NumberFieldLabel = React.forwardRef<
  React.ElementRef<typeof TextField.Label>,
  React.ComponentPropsWithoutRef<typeof TextField.Label>
>((props, ref) => {
  return <TextField.Label ref={ref} {...props} />
})

NumberFieldLabel.displayName = 'NumberField.Label'

/**
 * Main input field component that wraps TextField.Root.
 * Contains the actual input element and renders children (buttons, hints, errors) within TextField context.
 *
 * @param children - Components to render within the TextField (buttons, hints, errors)
 */
const NumberFieldField = React.forwardRef<
  React.ElementRef<typeof TextField.Root>,
  Omit<React.ComponentPropsWithoutRef<typeof TextField.Root>, 'value' | 'onChange' | 'name'> & {
    children?: React.ReactNode
  }
>((props, ref) => {
  const { value, disabled, name, onInputChange } = useNumberFieldContext()
  const { children, ...textFieldProps } = props

  return (
    <TextField.Root
      ref={ref}
      name={name}
      type="number"
      disabled={disabled}
      value={value.toString()}
      onChange={onInputChange}
      className="!grid !grid-cols-[max-content_1fr_max-content]"
      {...textFieldProps}
    >
      {children}
    </TextField.Root>
  )
})

NumberFieldField.displayName = 'NumberField.Field'

/**
 * Decrement button component that decreases the value by 1.
 * Automatically disabled when value reaches minimum.
 * Renders as a TextField.Slot on the left side with minus circle icon.
 */
const NumberFieldDecrementButton = React.forwardRef<
  HTMLButtonElement,
  Omit<React.ComponentPropsWithoutRef<typeof Button>, 'onClick'>
>((props, ref) => {
  const { value, min, disabled, onChange } = useNumberFieldContext()

  function onDecrement() {
    if (value > min) {
      onChange(value - 1)
    }
  }

  return (
    <TextField.Slot side="left">
      <Button
        ref={ref}
        type="button"
        variant="secondary"
        size="sm"
        onClick={onDecrement}
        disabled={disabled || value <= min}
        className="h-8 w-8 p-0"
        {...props}
      >
        <MinusCircleIcon className="h-4 w-4" />
      </Button>
    </TextField.Slot>
  )
})

NumberFieldDecrementButton.displayName = 'NumberField.DecrementButton'

/**
 * Increment button component that increases the value by 1.
 * Automatically disabled when value reaches maximum.
 * Renders as a TextField.Slot on the right side with plus circle icon.
 */
const NumberFieldIncrementButton = React.forwardRef<
  HTMLButtonElement,
  Omit<React.ComponentPropsWithoutRef<typeof Button>, 'onClick'>
>((props, ref) => {
  const { value, max, disabled, onChange } = useNumberFieldContext()

  function onIncrement() {
    if (value < max) {
      onChange(value + 1)
    }
  }

  return (
    <TextField.Slot side="right">
      <Button
        ref={ref}
        type="button"
        variant="secondary"
        size="sm"
        onClick={onIncrement}
        disabled={disabled || value >= max}
        className="h-8 w-8 p-0"
        {...props}
      >
        <PlusCircleIcon className="h-4 w-4" />
      </Button>
    </TextField.Slot>
  )
})

NumberFieldIncrementButton.displayName = 'NumberField.IncrementButton'

/**
 * Hint component for providing helpful text below the field.
 * Uses TextField.Hint from owly for consistent styling.
 * Must be used within NumberField.Field component.
 */
const NumberFieldHint = React.forwardRef<
  React.ElementRef<typeof TextField.Hint>,
  React.ComponentPropsWithoutRef<typeof TextField.Hint>
>((props, ref) => {
  return <TextField.Hint className="!col-span-full !row-start-2" ref={ref} {...props} />
})

NumberFieldHint.displayName = 'NumberField.Hint'

/**
 * Error component for displaying validation errors.
 * Uses TextField.Error from owly for consistent styling.
 * Must be used within NumberField.Field component.
 */
const NumberFieldError = React.forwardRef<
  React.ElementRef<typeof TextField.Error>,
  React.ComponentPropsWithoutRef<typeof TextField.Error>
>((props, ref) => {
  return <TextField.Error ref={ref} {...props} />
})

NumberFieldError.displayName = 'NumberField.Error'

export {
  NumberFieldRoot as Root,
  NumberFieldLabel as Label,
  NumberFieldField as Field,
  NumberFieldDecrementButton as DecrementButton,
  NumberFieldIncrementButton as IncrementButton,
  NumberFieldHint as Hint,
  NumberFieldError as Error,
}
