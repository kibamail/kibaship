import { createContext, Dispatch, SetStateAction, useContext } from 'react'
import { Application, Deployment } from '~/types'

export const ProjectPageContext = createContext<{
  selectedApplication: Application | null
  setSelectedApplication: Dispatch<SetStateAction<Application | null>>
  applicationsMap: Record<string, Application>
  selectedDeployment: Deployment | null
  setSelectedDeployment: Dispatch<SetStateAction<Deployment | null>>
}>({
  selectedApplication: null,
  setSelectedApplication() {},
  applicationsMap: {},
  selectedDeployment: null,
  setSelectedDeployment() {},
})

export function useProjectPageContext() {
  return useContext(ProjectPageContext)
}
