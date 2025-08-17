import { LogoutIcon } from '~/Components/Icons/logout.svg'
import { Link } from '@inertiajs/react'
import { Text } from '@kibamail/owly/text'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'

export function SignoutForm() {
  return (
    <DropdownMenu.Item className="rounded-lg">
      <Link
        href={'/w/logout'}
        method="post"
        as="button"
        className="p-2 flex w-full items-center hover:bg-(--background-secondary) rounded-lg cursor-pointer"
      >
        <LogoutIcon className="mr-1.5 w-5 h-5 kb-content-tertiary" />
        <Text>Sign out</Text>
      </Link>
    </DropdownMenu.Item>
  )
}
