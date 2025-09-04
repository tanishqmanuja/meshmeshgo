import { useEffect, useRef, useState } from 'react';
import { useGetList } from 'react-admin';
import { GraphCanvas, GraphCanvasRef } from 'reagraph';

type GraphNode = {
  id: string;
  label: string;
  fill: string;
};

type NetworkNode = {
  id: number;
  tag: string;
  is_local: boolean;
};

type GraphEdge = {
  id: string;
  source: string;
  target: string;
  label: string;
};

type NetworkLink = {
  id: string;
  from: number;
  to: number;
  weight: number;
};

export const NetworkGraph = () => {
  const ref = useRef<GraphCanvasRef | null>(null);
  const [nodes, setNodes] = useState<GraphNode[]>([]);
  const [edges, setEdges] = useState<GraphEdge[]>([]);
  useEffect(() => { ref.current?.fitNodesInView(); }, [nodes]);

  const { data: networkNodes } = useGetList<NetworkNode>('nodes', { });
  const { data: networkLinks, total, isPending, error, refetch: refetchLinks } = useGetList<NetworkLink>('links', { }, { enabled: false });

  useEffect(() => {
    if (networkNodes) {
      const nodes = networkNodes.map((node: any) => ({
        id: node.id.toString(),
        label: node.tag,
        fill: node.is_local ? 'yellow' : 'blue'
      }));
      console.log('useEffect.nodes', nodes, edges);
      setNodes(nodes);
      refetchLinks();
    }
  }, [networkNodes]);


  useEffect(() => {
    if (networkLinks) {
      const edges = networkLinks.map((edge: any) => ({
        id: edge.id.toString(),
        source: edge.from.toString(),
        target: edge.to.toString(),
        label: edge.weight.toString(),
        weight: edge.weight
      }));
      //console.log('useEffect.nodes', ref.current?.getGraph().nodes());
      console.log('useEffect.edges', edges);
      setEdges(edges);
    }
  }, [networkLinks]);

  return (
    <GraphCanvas ref={ref} nodes={nodes} edges={edges} cameraMode="rotate" layoutType='forceDirected3d' />
  );
};