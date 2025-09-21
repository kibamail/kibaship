import { PropsWithChildren } from 'react'
import AuthenticatedLayout from './AuthenticatedLayout'
import { Link, InertiaLinkProps, usePage } from '@inertiajs/react'

import cn from 'classnames'
import { K8sIcon } from '~/Components/Icons/k8s.svg'
import { PageProps } from '~/types'
import { CloudCheckIcon } from '~/Components/Icons/cloud-check.svg'

interface SubmenuItemLinkProps extends React.PropsWithChildren, InertiaLinkProps {
  isActive?: boolean
}

export function SubmenuItemLink({ children, isActive, ...linkProps }: SubmenuItemLinkProps) {
  return (
    <Link
      data-active={isActive}
      href=""
      className={cn(
        'w-full px-2 h-10 border-t border-l border-r border-b-2 rounded-lg gap-x-2 flex items-center  transition-[background] ease-in-out group text-sm',
        {
          'bg-(--background-primary) kb-border-tertiary shadow-[0px_1px_0px_0px_var(--black-5)] kb-content-primary [&>span]:text-(--content-primary) [&>svg]:text-(--content-primary)':
            isActive,
          'hover:bg-(--background-hover) border-transparent text-owly-content-secondary': !isActive,
        }
      )}
      {...linkProps}
    >
      {children}
    </Link>
  )
}

function SidebarItems() {
  const {
    url,
    props: { workspace },
  } = usePage<PageProps>()

  function getSidebarItems() {
    if (url.includes('cloud')) {
      return [
        {
          title: 'Clusters',
          href: '/cloud/clusters',
          icon: <K8sIcon className="w-5" />,
          isActive: url.includes('cloud/clusters'),
        },
        {
          title: 'Cloud providers',
          href: '/cloud/providers',
          icon: <CloudCheckIcon className="w-4" />,
          isActive: url.includes('cloud/providers'),
        },
      ]
    }
  }

  const items = getSidebarItems()

  return items?.map((item) => (
    <SubmenuItemLink
      key={item?.href}
      isActive={item?.isActive}
      href={`/w/${workspace.slug}${item?.href}`}
    >
      {item?.icon}
      {item?.title}
    </SubmenuItemLink>
  ))
}

export function SidebarPageLayout({ children }: PropsWithChildren) {
  return (
    <AuthenticatedLayout>
      <div className="w-full flex flex-grow h-full">
        <div className="w-[360px] h-full border-r border-owly-border-tertiary px-4 py-8">
          <div className="flex flex-col">
            <p className="text-xs uppercase text-owly-content-tertiary font-medium">Clusters</p>

            <div className="mt-4 flex flex-col gap-2">
              <SidebarItems />
            </div>
          </div>
        </div>
        <div className="w-full flex-grow h-full">
          <div className="px-4 lg:px-0 max-w-4xl mx-auto py-8">{children}</div>
        </div>
      </div>
    </AuthenticatedLayout>
  )
}
