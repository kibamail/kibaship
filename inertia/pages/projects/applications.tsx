'use client'

import { Head, usePage, router } from '@inertiajs/react'
import AuthenticatedLayout from '~/Layouts/AuthenticatedLayout'
import { ReactFlow, Background, Controls } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useMemo, useState } from 'react'
import { Button } from '@kibamail/owly'
import { PlusIcon } from '~/Components/Icons/plus.svg'
import { NewApplicationCommand } from '~/Components/Applications/NewApplicationCommand'
import { GitRepositoryAppNode } from '~/Components/Projects/Nodes/GitRepositoryAppNode'
import { ProjectSettings } from '~/Components/Projects/ProjectSettings'
import { Application, Deployment, PageProps, Project } from '~/types'
import { ProjectPageContext } from './projects-context'
import { NoApplicationsInProject } from '~/Components/Dashboard/NoApplications'

const initialEdges = [{ id: 'n1-n2', source: 'n1', target: 'n2' }]

const nodeTypes = {
  // 'mysql' | 'postgres' | 'git' | 'docker_image' | 'template'
  git: GitRepositoryAppNode,
}

export default function ApplicationsPage() {
  const { props, url } = usePage<
    PageProps & {
      project: Project
    }
  >()
  const urlObject = new URL(url, 'https://dummy.com')

  function applicationClickUrl(applicationId: string) {
    const currentUrl = typeof window === 'undefined' ? urlObject : new URL(window.location.href)
    currentUrl.searchParams.set('application', applicationId)
    currentUrl.searchParams.set('tab', 'deployments')

    return `/w/${props.workspace.slug}/applications?${currentUrl.searchParams.toString()}`
  }

  const [nodes] = useState(() => {
    return (
      props?.applications?.map((application, idx) => ({
        id: application.id,
        type: application.type,
        position: { x: idx * 400, y: 0 },
        data: {
          label: application.name,
          getApplicationUrl: () => applicationClickUrl(application.id),
        },
      })) || []
    )
  })

  const [edges] = useState(initialEdges)
  const [newApplicationCommandOpen, setNewApplicationCommandOpen] = useState(false)

  const selectedApplicationId = urlObject.searchParams.get('application') as string

  const applicationsMap = useMemo(() => {
    return (
      props?.applications?.reduce(
        (acc, application) => {
          acc[application.id] = application
          return acc
        },
        {} as Record<string, Application>
      ) || {}
    )
  }, [props?.applications])

  const [selectedApplication, setSelectedApplication] = useState<Application | null>(
    applicationsMap[selectedApplicationId] || null
  )
  const [selectedDeployment, setSelectedDeployment] = useState<Deployment | null>(null)

  function onSelectedApplication() {
    const currentUrl = new URL(window.location.href)
    currentUrl.searchParams.delete('application')

    router.visit(`/w/${props.workspace.slug}/applications/${selectedApplication?.id}`)
  }

  function onSelectedDeployment() {
    const currentUrl = new URL(window.location.href)
    currentUrl.searchParams.delete('deployment')

    window.history.replaceState({}, '', currentUrl)
  }

  return (
    <AuthenticatedLayout>
      <Head title="Applications" />

      {!props?.applications?.length ? (
        <div className="px-4 lg:px-0 max-w-6xl mx-auto">
          <NoApplicationsInProject />
        </div>
      ) : (
        <ProjectPageContext.Provider
          value={{
            selectedApplication,
            setSelectedApplication,
            applicationsMap,
            selectedDeployment,
            setSelectedDeployment(deployment) {
              setSelectedDeployment(deployment)

              onSelectedDeployment()
            },
          }}
        >
          <div className="w-full h-full relative">
            <div className="absolute top-0 right-0 p-4 z-20">
              <Button onClick={() => setNewApplicationCommandOpen(true)}>
                <PlusIcon className="!size-5" />
                New application
              </Button>
              <NewApplicationCommand
                open={newApplicationCommandOpen}
                onOpenChange={setNewApplicationCommandOpen}
              />
            </div>
            <ReactFlow nodes={nodes} edges={edges} nodeTypes={nodeTypes} fitView>
              <Background />
              <Controls />
            </ReactFlow>

            <ProjectSettings
              isOpen={selectedApplication !== null}
              onOpenChange={() => {
                setSelectedApplication(null)

                // Wait until modal animation is done.
                setTimeout(() => {
                  onSelectedApplication()
                }, 300)
              }}
            />
          </div>
        </ProjectPageContext.Provider>
      )}
    </AuthenticatedLayout>
  )
}
