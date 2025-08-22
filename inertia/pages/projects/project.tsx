'use client'

import { Head, usePage, router } from '@inertiajs/react'
import AuthenticatedLayout from '~/Layouts/AuthenticatedLayout'
import { ReactFlow, Background, Controls } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useEffect, useMemo, useState } from 'react'
import { Button } from '@kibamail/owly'
import { PlusIcon } from '~/Components/Icons/plus.svg'
import { NewApplicationCommand } from '~/Components/Applications/NewApplicationCommand'
import { GitRepositoryAppNode } from '~/Components/Projects/Nodes/GitRepositoryAppNode'
import { ProjectSettings } from '~/Components/Projects/ProjectSettings'
import { Application, PageProps, Project } from '~/types'
import { ProjectPageContext } from './projects-context'
import { VaulDrawer } from '~/Components/VaulSheet'

const initialEdges = [{ id: 'n1-n2', source: 'n1', target: 'n2' }]

const nodeTypes = {
  // 'mysql' | 'postgres' | 'git' | 'docker_image' | 'template'
  git: GitRepositoryAppNode,
}

export default function ProjectPage() {
  const { props, url } = usePage<
    PageProps & {
      project: Project
    }
  >()
  const urlObject = new URL(url, 'https://dummy.com')

  function applicationClickUrl(applicationId: string) {
    urlObject.searchParams.set('application', applicationId)
    urlObject.searchParams.set('tab', 'deployments')

    return `/w/${props.workspace.slug}/p/${props.project.id}?${urlObject.searchParams.toString()}`
  }

  const [nodes] = useState(() => {
    return (
      props.project?.applications?.map((application) => ({
        id: application.id,
        type: application.type,
        position: { x: 0, y: 0 },
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
    return props.project?.applications?.reduce(
      (acc, application) => {
        acc[application.id] = application
        return acc
      },
      {} as Record<string, Application>
    )
  }, [props.project?.applications])

  const [selectedApplication, setSelectedApplication] = useState<Application | null>(
    applicationsMap[selectedApplicationId] || null
  )

  function onSelectedApplication() {
    urlObject.searchParams.delete('application')

    router.visit(
      `/w/${props.workspace.slug}/p/${props.project.id}?${urlObject.searchParams.toString()}`
    )
  }

  return (
    <AuthenticatedLayout>
      <Head title="Project" />

      <ProjectPageContext.Provider
        value={{
          selectedApplication,
          setSelectedApplication,
          applicationsMap,
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
    </AuthenticatedLayout>
  )
}
