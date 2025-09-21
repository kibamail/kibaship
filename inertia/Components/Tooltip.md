# Tooltip Component

A flexible tooltip component built on top of `@radix-ui/react-tooltip` that follows the Owly design system.

## Features

- **Simple API**: Just wrap any element and provide content
- **Rich Content**: Supports React elements as tooltip content
- **Flexible Positioning**: Configurable side, alignment, and offset
- **Performance Optimized**: Provider pattern for multiple tooltips
- **Accessible**: Built on Radix UI primitives with full accessibility support
- **Consistent Styling**: Uses Owly design tokens for colors and spacing

## Basic Usage

### Simple Tooltip

```tsx
import { Tooltip } from '~/Components/Tooltip'
import { Button } from '@kibamail/owly/button'

function MyComponent() {
  return (
    <Tooltip content="This is a helpful tooltip">
      <Button>Hover me</Button>
    </Tooltip>
  )
}
```

### Rich Content

```tsx
<Tooltip 
  content={
    <div>
      <div className="font-semibold">Rich Content</div>
      <div className="text-xs opacity-90">This tooltip contains HTML</div>
    </div>
  }
>
  <Button>Rich tooltip</Button>
</Tooltip>
```

### Positioning

```tsx
<Tooltip content="Tooltip on the right" side="right">
  <Button>Right tooltip</Button>
</Tooltip>

<Tooltip content="Tooltip on bottom" side="bottom" align="start">
  <Button>Bottom left tooltip</Button>
</Tooltip>
```

## Multiple Tooltips (Optimized)

When you have multiple tooltips in a component tree, use `TooltipProvider` and `SimpleTooltip` for better performance:

```tsx
import { TooltipProvider, SimpleTooltip } from '~/Components/Tooltip'

function MyComponent() {
  return (
    <TooltipProvider>
      <div className="flex gap-4">
        <SimpleTooltip content="First tooltip">
          <Button>Button 1</Button>
        </SimpleTooltip>
        
        <SimpleTooltip content="Second tooltip">
          <Button>Button 2</Button>
        </SimpleTooltip>
        
        <SimpleTooltip content="Third tooltip">
          <Button>Button 3</Button>
        </SimpleTooltip>
      </div>
    </TooltipProvider>
  )
}
```

## API Reference

### Tooltip Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `children` | `React.ReactNode` | - | The element that triggers the tooltip |
| `content` | `React.ReactNode` | - | The content to display in the tooltip |
| `side` | `'top' \| 'right' \| 'bottom' \| 'left'` | `'top'` | Which side of the trigger to show the tooltip |
| `align` | `'start' \| 'center' \| 'end'` | `'center'` | How to align the tooltip relative to the trigger |
| `sideOffset` | `number` | `8` | Distance in pixels from the trigger |
| `delayDuration` | `number` | `400` | Delay in ms before showing tooltip |
| `skipDelayDuration` | `number` | `300` | Time in ms to skip delay when moving between tooltips |
| `open` | `boolean` | - | Controlled open state |
| `defaultOpen` | `boolean` | - | Default open state (uncontrolled) |
| `onOpenChange` | `(open: boolean) => void` | - | Callback when open state changes |
| `className` | `string` | - | Additional CSS classes for the tooltip content |

### TooltipProvider Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `children` | `React.ReactNode` | - | Child components |
| `delayDuration` | `number` | `400` | Global delay duration for all tooltips |
| `skipDelayDuration` | `number` | `300` | Global skip delay duration |

### SimpleTooltip Props

Same as `Tooltip` props except `delayDuration` and `skipDelayDuration` (inherited from provider).

## Styling

The tooltip uses the following Owly design tokens:

- **Background**: `--color-owly-background-inverse` (dark background)
- **Text Color**: `--color-owly-content-primary-inverse` (light text)
- **Padding**: `8px` horizontal, `10px` vertical
- **Border Radius**: `8px`

You can override styles using the `className` prop:

```tsx
<Tooltip 
  content="Custom styled tooltip" 
  className="!bg-blue-600 !text-white !px-4 !py-3"
>
  <Button>Custom style</Button>
</Tooltip>
```

## Examples

See `TooltipExample.tsx` for comprehensive usage examples including:

- Basic tooltips with different positioning
- Rich HTML content
- Multiple tooltips with provider
- Custom styling
- Non-button elements

## Accessibility

The component inherits full accessibility support from Radix UI:

- Proper ARIA attributes
- Keyboard navigation support
- Screen reader compatibility
- Focus management

## Performance Notes

- Use `TooltipProvider` + `SimpleTooltip` when you have multiple tooltips
- Each `Tooltip` component creates its own provider, which is fine for single tooltips
- The component uses Radix UI's optimized portal rendering
- Animations are CSS-based for smooth performance
