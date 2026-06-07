/** 将多门课程按主题族归并为「星座」，并在圆周上分区排布（无虚拟 hub 节点） */

export interface ConstellationGroup {
  key: string
  label: string
  domainIds: string[]
  /** 该星座内全部领域节点总数（含 domain / module / topic） */
  nodeCount: number
}

export interface DomainConstellationInput {
  domainId: string
  name: string
  slug?: string
  nodeCount?: number
}

const CONSTELLATION_LABELS: Record<string, string> = {
  go: 'Go 语言',
  python: 'Python',
  rust: 'Rust',
  agent: 'Agent 开发',
  math: '数学',
  trade: '外贸',
}

/** 与后端 domain.TopicRoot 对齐的粗粒度主题根 */
export function topicRootKey(slug: string, name: string): string {
  const s = (slug || slugifyLoose(name)).toLowerCase().trim()
  if (!s) return 'other'
  if (s === 'go' || s === 'golang' || s === 'go-language' || s.startsWith('go-')) return 'go'
  if (s === 'python' || s === 'py' || s.startsWith('python')) return 'python'
  if (s === 'rust' || s.startsWith('rust')) return 'rust'
  if (s.includes('agent')) return 'agent'
  if (s.includes('math') || name.includes('数学')) return 'math'
  if (s.includes('trade') || name.includes('外贸')) return 'trade'
  const head = s.split('-')[0]
  return head || s
}

function slugifyLoose(name: string): string {
  return name
    .trim()
    .toLowerCase()
    .replace(/\s+/g, '-')
    .replace(/[^a-z0-9\u4e00-\u9fff-]/g, '')
}

export function constellationLabel(key: string, sampleName?: string): string {
  if (CONSTELLATION_LABELS[key]) return CONSTELLATION_LABELS[key]
  if (sampleName?.trim()) return sampleName.trim()
  return key
}

export function groupDomainsIntoConstellations(domains: DomainConstellationInput[]): ConstellationGroup[] {
  const buckets = new Map<string, { names: string[]; domainIds: string[]; nodeCount: number }>()
  for (const d of domains) {
    const key = topicRootKey(d.slug ?? '', d.name)
    const bucket = buckets.get(key) ?? { names: [], domainIds: [], nodeCount: 0 }
    bucket.names.push(d.name)
    bucket.domainIds.push(d.domainId)
    bucket.nodeCount += Math.max(1, d.nodeCount ?? 1)
    buckets.set(key, bucket)
  }
  return [...buckets.entries()].map(([key, bucket]) => ({
    key,
    label: constellationLabel(key, bucket.names[0]),
    domainIds: bucket.domainIds,
    nodeCount: bucket.nodeCount,
  }))
}

/** 跨星座斥力边长度：节点越多，与其他星座拉得越远 */
export function constellationSeparationLength(groupA: ConstellationGroup, groupB: ConstellationGroup): number {
  if (groupA.key === groupB.key) {
    const perDomain = groupA.nodeCount / Math.max(1, groupA.domainIds.length)
    return 260 + perDomain * 6
  }
  const mass = groupA.nodeCount + groupB.nodeCount
  const domainSpread = groupA.domainIds.length + groupB.domainIds.length
  return 2150 + mass * 108 + Math.max(0, domainSpread - 2) * 180
}

/** 多领域排布：星座质心按节点规模占扇区并拉开，同星座内课程紧密成簇 */
export function layoutDomainCentersByConstellation(groups: ConstellationGroup[]): Map<string, { x: number; y: number }> {
  const out = new Map<string, { x: number; y: number }>()
  const totalDomains = groups.reduce((sum, g) => sum + g.domainIds.length, 0)
  const totalNodes = groups.reduce((sum, g) => sum + g.nodeCount, 0)
  if (totalDomains === 0) return out
  if (totalDomains === 1) {
    out.set(groups[0]!.domainIds[0]!, { x: 0, y: 0 })
    return out
  }

  const groupCount = groups.length
  const baseRadius = 1080 + Math.max(0, totalNodes - 6) * 64 + Math.max(0, groupCount - 2) * 360

  const placeCluster = (group: ConstellationGroup, centroidAngle: number) => {
    const r = baseRadius + group.nodeCount * 52
    const cx = r * Math.cos(centroidAngle)
    const cy = r * Math.sin(centroidAngle)
    const n = group.domainIds.length
    const clusterRadius = n <= 1 ? 0 : 75 + n * 18

    group.domainIds.forEach((id, i) => {
      if (n === 1) {
        out.set(id, { x: cx, y: cy })
        return
      }
      const localAngle = (2 * Math.PI * i) / n - Math.PI / 2
      out.set(id, {
        x: cx + clusterRadius * Math.cos(localAngle),
        y: cy + clusterRadius * Math.sin(localAngle),
      })
    })
  }

  if (groupCount === 1) {
    placeCluster(groups[0]!, -Math.PI / 2)
    return out
  }

  const gapTotal = Math.min(1.55, 0.36 * groupCount)
  const usable = 2 * Math.PI - gapTotal
  const gapEach = gapTotal / groupCount
  let cursor = -Math.PI / 2 + gapEach / 2

  for (const group of groups) {
    const domainWeight = group.domainIds.length / totalDomains
    const nodeWeight = group.nodeCount / Math.max(1, totalNodes)
    const weight = domainWeight * 0.35 + nodeWeight * 0.65
    const sectorSpan = usable * weight
    placeCluster(group, cursor + sectorSpan / 2)
    cursor += sectorSpan + gapEach
  }
  return out
}
