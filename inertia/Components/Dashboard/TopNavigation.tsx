import { Link, usePage } from '@inertiajs/react'
import * as Tabs from '@kibamail/owly/tabs'
import { CloudCheckIcon } from '../Icons/cloud-check.svg'
import { BoxIcon } from '../Icons/box.svg'
import { StackIcon } from '../Icons/stack.svg'
import { ClusterIcon } from '../Icons/cluster.svg'
import { GitIcon } from '../Icons/git.svg'
import { PageProps } from '~/types'

export function TopNavigation() {
  const pageProps = usePage<PageProps>()

  const allWorkspaces = pageProps?.props?.profile?.workspaces

  const activeWorkspace = allWorkspaces.find((workspace) =>
    pageProps?.url?.includes(`/${workspace.slug}`)
  )

  function getWorkspacePath(path: string) {
    return `/w/${activeWorkspace?.slug}/${path}`
  }

  function getActiveTab() {
    if (pageProps?.url?.includes('dashboard')) {
      return 'dashboard'
    }

    if (pageProps?.url?.includes('applications')) {
      return 'applications'
    }

    if (pageProps?.url?.includes('cloud')) {
      return 'cloud'
    }

    if (pageProps?.url?.includes('integrations')) {
      return 'integrations'
    }

    if (pageProps?.url?.includes('monitoring')) {
      return 'monitoring'
    }

    return 'dashboard'
  }

  return (
    <div className="w-full h-9 px-2">
      <Tabs.Root variant="secondary" value={getActiveTab()}>
        <Tabs.List className="!border-b-0">
          <Tabs.Trigger value="dashboard" asChild>
            <Link href={getWorkspacePath('dashboard')}>
              <BoxIcon />
              Dashboard
            </Link>
          </Tabs.Trigger>
          <Tabs.Trigger value="applications" asChild>
            <Link href={getWorkspacePath('applications')}>
              <StackIcon />
              Applications
            </Link>
          </Tabs.Trigger>
          <Tabs.Trigger value="cloud" asChild>
            <Link href={getWorkspacePath('cloud/clusters')}>
              <CloudCheckIcon />
              Cloud
            </Link>
          </Tabs.Trigger>

          <Tabs.Trigger value="integrations" asChild>
            <Link href={getWorkspacePath('integrations')}>
              <GitIcon />
              Integrations
            </Link>
          </Tabs.Trigger>
          <Tabs.Trigger value="monitoring" asChild>
            <Link href={getWorkspacePath('monitoring')}>
              <ClusterIcon />
              Monitoring
            </Link>
          </Tabs.Trigger>
          <Tabs.Indicator />
        </Tabs.List>
      </Tabs.Root>
    </div>
  )
}
