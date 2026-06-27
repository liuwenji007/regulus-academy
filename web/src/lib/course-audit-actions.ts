import {
  submitDomainAuditJob,
  submitDomainOptimizeJob,
  pollDomainAuditJob,
  pollDomainOptimizeJob,
  ApiError,
  type CourseAuditReport,
} from './api'
import { tryRecoverFromQuotaError } from './cloud'
import {
  applyServerBuildProgress,
  clearPendingBuild,
  finishDomainBuildJobError,
  finishDomainBuildJobSuccess,
  savePendingBuild,
  tryStartDomainBuildJob,
} from './domain-build-job'
import { showCourseOptimizeDiff } from '../components/course-optimize-diff'

export async function handleDomainAudit(
  domainId: string,
  domainName: string,
  onProgress?: (message: string) => void
): Promise<{ report: CourseAuditReport; auditJobId: string }> {
  if (!tryStartDomainBuildJob(domainName, { kind: 'audit', domainId })) {
    throw new ApiError('已有课程任务正在进行，请稍候')
  }
  try {
    const { jobId } = await submitDomainAuditJob(domainId)
    savePendingBuild({ jobId, topic: domainName, kind: 'audit', domainId })
    const report = await pollDomainAuditJob(jobId, (status) => {
      applyServerBuildProgress(status)
      onProgress?.(status.message)
    })
    clearPendingBuild()
    finishDomainBuildJobSuccess({ domainId, message: '体检报告已生成' })
    return { report, auditJobId: jobId }
  } catch (e) {
    clearPendingBuild()
    if (await tryRecoverFromQuotaError(e)) {
      finishDomainBuildJobError('已取消或已配置 Key')
      throw e
    }
    const msg = e instanceof ApiError ? e.message : '课程体检失败'
    finishDomainBuildJobError(msg)
    throw e
  }
}

export async function handleDomainOptimize(
  domainId: string,
  domainName: string,
  findingIds: string[],
  auditJobId: string,
  onApplied: (message?: string) => void,
  onProgress?: (message: string) => void
): Promise<void> {
  if (!tryStartDomainBuildJob(domainName, { kind: 'optimize', domainId })) {
    throw new ApiError('已有课程任务正在进行，请稍候')
  }
  try {
    const { jobId } = await submitDomainOptimizeJob(domainId, findingIds, auditJobId)
    savePendingBuild({ jobId, topic: domainName, kind: 'optimize', domainId })
    const patch = await pollDomainOptimizeJob(jobId, (status) => {
      applyServerBuildProgress(status)
      onProgress?.(status.message)
    })
    clearPendingBuild()
    finishDomainBuildJobSuccess({ domainId, message: '优化方案已生成' })
    if (!patch.patches?.length) {
      onApplied('没有生成可应用的补丁，请尝试其他建议项')
      return
    }
    showCourseOptimizeDiff({
      domainId,
      jobId,
      patch,
      onApplied,
    })
  } catch (e) {
    clearPendingBuild()
    if (await tryRecoverFromQuotaError(e)) {
      finishDomainBuildJobError('已取消或已配置 Key')
      throw e
    }
    const msg = e instanceof ApiError ? e.message : '课程优化失败'
    finishDomainBuildJobError(msg)
    throw e
  }
}
