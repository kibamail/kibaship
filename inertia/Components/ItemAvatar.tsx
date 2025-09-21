import classNames from 'classnames'

interface TeamAvatarProps {
  size?: 'sm' | 'md'
  name?: string
  className?: string
  image?: string
}

export function ItemAvatar({ size = 'md', name, className, image }: TeamAvatarProps) {
  const classes = classNames(
    'mr-1.5 text-sm shadow-[0px_0px_0px_1px_rgba(0,0,0,0.10)_inset] kb-background-info rounded-lg flex items-center justify-center kb-content-primary-inverse uppercase',
    {
      'w-5 h-5': size === 'sm',
      'w-6 h-6': size === 'md',
    }
  )

  if (image) {
    return <img src={image} alt={name} className={classNames(classes, className)} />
  }

  return (
    <span className={classNames(classes, getTeamAvatarBackgroundColor(name?.[0] ?? ''), className)}>
      {name?.[0]}
    </span>
  )
}

function getTeamAvatarBackgroundColor(firstCharacter: string) {
  const colors = [
    'kb-background-info',
    'kb-background-positive',
    'kb-background-negative',
    'kb-background-warning',
    'kb-background-highlight',
  ]

  const asciiValue = firstCharacter.charCodeAt(0)
  const index = (asciiValue - 97) % colors.length

  return colors[index] ?? colors?.[0]
}
