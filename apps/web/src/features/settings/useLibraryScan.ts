import { useCallback, useEffect, useRef, useState } from 'react'
import type { Job, Library } from '../../api/types'
import { api } from '../../api/client'

const terminalStatuses = new Set<Job['status']>(['succeeded', 'failed'])

export function useLibraryScan(onCompleted: () => void | Promise<void>) {
  const [jobs, setJobs] = useState<Record<string, Job>>({})
  const completedJobs = useRef(new Set<string>())

  const startScan = useCallback(async (library: Library) => {
    const job = await api.libraries.scan(library.id)
    setJobs((current) => ({ ...current, [library.id]: job }))
    return job
  }, [])

  useEffect(() => {
    const activeEntries = Object.entries(jobs).filter(([, job]) => !terminalStatuses.has(job.status))
    if (activeEntries.length === 0) return

    let cancelled = false
    const poll = async () => {
      const updates = await Promise.all(
        activeEntries.map(async ([libraryId, job]) => {
          try {
            return [libraryId, await api.jobs.get(job.id)] as const
          } catch {
            return [libraryId, job] as const
          }
        }),
      )
      if (cancelled) return

      const newlyFinished = updates.filter(([, job]) => terminalStatuses.has(job.status) && !completedJobs.current.has(job.id))
      for (const [, job] of newlyFinished) completedJobs.current.add(job.id)
      setJobs((current) => {
        const next = { ...current }
        for (const [libraryId, job] of updates) next[libraryId] = job
        return next
      })
      if (newlyFinished.length > 0) void onCompleted()
    }

    const timer = window.setInterval(() => void poll(), 850)
    void poll()
    return () => {
      cancelled = true
      window.clearInterval(timer)
    }
  }, [jobs, onCompleted])

  return { jobs, startScan }
}
