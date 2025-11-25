import { useEffect, useRef, useState } from 'react'
import * as d3 from 'd3'
import axios from 'axios'

const API_URL = 'http://localhost:8080/api'

interface Node {
    id: string
    title: string
}

interface Link {
    source_id: string
    target_id: string
    source?: any
    target?: any
}

interface GraphViewProps {
    onClose: () => void
}

export default function GraphView({ onClose }: GraphViewProps) {
    const svgRef = useRef<SVGSVGElement>(null)
    const [stats, setStats] = useState<any>(null)
    const [loading, setLoading] = useState(true)

    useEffect(() => {
        fetchAndRenderGraph()
    }, [])

    const fetchAndRenderGraph = async () => {
        try {
            const res = await axios.get(`${API_URL}/graph`)
            setStats(res.data.stats)
            renderGraph(res.data.links || [])
        } catch (error) {
            console.error('Failed to fetch graph:', error)
        } finally {
            setLoading(false)
        }
    }

    const renderGraph = (links: Link[]) => {
        if (!svgRef.current || links.length === 0) return

        // Extract nodes from links
        const nodeMap = new Map<string, Node>()
        links.forEach((link: any) => {
            if (link.source && link.source.id && !nodeMap.has(link.source.id)) {
                nodeMap.set(link.source.id, { id: link.source.id, title: link.source.title })
            }
            if (link.target && link.target.id && !nodeMap.has(link.target.id)) {
                nodeMap.set(link.target.id, { id: link.target.id, title: link.target.title })
            }
        })
        const nodes = Array.from(nodeMap.values())

        // Setup SVG
        const width = window.innerWidth
        const height = window.innerHeight - 100

        const svg = d3.select(svgRef.current)
            .attr('width', width)
            .attr('height', height)

        svg.selectAll('*').remove()

        // Create force simulation
        const simulation = d3.forceSimulation(nodes as any)
            .force('link', d3.forceLink(links)
                .id((d: any) => d.id)
                .distance(150))
            .force('charge', d3.forceManyBody().strength(-400))
            .force('center', d3.forceCenter(width / 2, height / 2))
            .force('collision', d3.forceCollide().radius(30))

        // Create links
        const link = svg.append('g')
            .selectAll('line')
            .data(links)
            .enter().append('line')
            .style('stroke', '#2a2a2a')
            .style('stroke-width', 2)

        // Create nodes
        const node = svg.append('g')
            .selectAll('circle')
            .data(nodes)
            .enter().append('circle')
            .attr('r', 12)
            .style('fill', '#ff8935')
            .style('stroke', '#0d0d0d')
            .style('stroke-width', 2)
            .style('cursor', 'pointer')
            .call(d3.drag<any, any>()
                .on('start', dragstarted)
                .on('drag', dragged)
                .on('end', dragended) as any)
            .on('click', (_event, d: any) => {
                window.location.href = `/document/${d.id}`
            })
            .on('mouseenter', function () {
                d3.select(this)
                    .transition()
                    .duration(200)
                    .attr('r', 16)
                    .style('filter', 'drop-shadow(0 0 10px #ff8935)')
            })
            .on('mouseleave', function () {
                d3.select(this)
                    .transition()
                    .duration(200)
                    .attr('r', 12)
                    .style('filter', 'none')
            })

        // Add labels
        const label = svg.append('g')
            .selectAll('text')
            .data(nodes)
            .enter().append('text')
            .text((d: Node) => d.title || 'Untitled')
            .style('font-size', '12px')
            .style('fill', '#e0e0e0')
            .style('pointer-events', 'none')
            .attr('text-anchor', 'middle')

        // Update positions on tick
        simulation.on('tick', () => {
            link
                .attr('x1', (d: any) => d.source.x)
                .attr('y1', (d: any) => d.source.y)
                .attr('x2', (d: any) => d.target.x)
                .attr('y2', (d: any) => d.target.y)

            node
                .attr('cx', (d: any) => d.x)
                .attr('cy', (d: any) => d.y)

            label
                .attr('x', (d: any) => d.x)
                .attr('y', (d: any) => d.y + 28)
        })

        function dragstarted(event: any) {
            if (!event.active) simulation.alphaTarget(0.3).restart()
            event.subject.fx = event.subject.x
            event.subject.fy = event.subject.y
        }

        function dragged(event: any) {
            event.subject.fx = event.x
            event.subject.fy = event.y
        }

        function dragended(event: any) {
            if (!event.active) simulation.alphaTarget(0)
            event.subject.fx = null
            event.subject.fy = null
        }
    }

    return (
        <div className="graph-overlay">
            <div className="graph-header">
                <div style={{ display: 'flex', alignItems: 'center', gap: '24px' }}>
                    <h2 style={{ color: 'var(--accent-primary)', margin: 0 }}>Knowledge Graph</h2>
                    {stats && (
                        <div style={{
                            display: 'flex',
                            gap: '16px',
                            fontSize: '14px',
                            color: 'var(--text-secondary)'
                        }}>
                            <span>📄 {stats.total_documents} documents</span>
                            <span>🔗 {stats.total_links} links</span>
                            <span>📊 {stats.avg_degree.toFixed(1)} avg connections</span>
                        </div>
                    )}
                </div>

                <button className="btn" onClick={onClose}>
                    ✕ Close
                </button>
            </div>

            <div className="graph-canvas">
                {loading ? (
                    <div className="loading" style={{ height: '100%' }}>
                        <div className="spinner"></div>
                        <span style={{ marginLeft: '12px' }}>Loading graph...</span>
                    </div>
                ) : (
                    <svg ref={svgRef}></svg>
                )}
            </div>
        </div>
    )
}
