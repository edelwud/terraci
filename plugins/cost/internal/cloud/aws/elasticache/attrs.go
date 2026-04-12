package elasticache

import "github.com/edelwud/terraci/plugins/cost/internal/costutil"

type clusterAttrs struct {
	NodeType              string
	Engine                string
	NumCacheNodes         int
	SnapshotRetentionDays int
}

func parseClusterAttrs(attrs map[string]any) clusterAttrs {
	return clusterAttrs{
		NodeType:              costutil.GetStringAttr(attrs, "node_type"),
		Engine:                costutil.GetStringAttr(attrs, "engine"),
		NumCacheNodes:         costutil.GetIntAttr(attrs, "num_cache_nodes"),
		SnapshotRetentionDays: costutil.GetIntAttr(attrs, "snapshot_retention_limit"),
	}
}

type replicationGroupAttrs struct {
	NodeType              string
	NumNodeGroups         int
	NumNodeGroupsSet      bool
	ReplicasPerNodeGroup  int
	NumberCacheClusters   int
	SnapshotRetentionDays int
}

func parseReplicationGroupAttrs(attrs map[string]any) replicationGroupAttrs {
	numNodeGroups := costutil.GetIntAttr(attrs, "num_node_groups")
	parsed := replicationGroupAttrs{
		NodeType:              costutil.GetStringAttr(attrs, "node_type"),
		NumNodeGroups:         numNodeGroups,
		NumNodeGroupsSet:      numNodeGroups != 0,
		ReplicasPerNodeGroup:  costutil.GetIntAttr(attrs, "replicas_per_node_group"),
		NumberCacheClusters:   costutil.GetIntAttr(attrs, "number_cache_clusters"),
		SnapshotRetentionDays: costutil.GetIntAttr(attrs, "snapshot_retention_limit"),
	}
	if parsed.NumNodeGroups == 0 {
		parsed.NumNodeGroups = 1
	}
	return parsed
}

func (a replicationGroupAttrs) totalNodes() int {
	total := a.NumNodeGroups * (1 + a.ReplicasPerNodeGroup)
	if total == 1 && a.NumberCacheClusters > 0 {
		total = a.NumberCacheClusters
	}
	return total
}

type serverlessAttrs struct {
	Engine       string
	StorageMaxGB float64
}

func parseServerlessAttrs(attrs map[string]any) serverlessAttrs {
	limits := costutil.GetFirstObjectAttr(attrs, "cache_usage_limits")
	storage := costutil.GetFirstObjectAttr(limits, "data_storage")
	return serverlessAttrs{
		Engine:       costutil.GetStringAttr(attrs, "engine"),
		StorageMaxGB: costutil.GetFloatAttr(storage, "maximum"),
	}
}
