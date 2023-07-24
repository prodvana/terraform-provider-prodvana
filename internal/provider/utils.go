package provider

import (
	"context"
	"time"

	"github.com/pkg/errors"
	env_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/environment"
)

func WaitForClusterWithTimeout(ctx context.Context, client env_pb.EnvironmentManagerClient, clusterId, clusterName, timeoutDuration string) error {
	// keep checking to see if linking succeeded until timeout
	timeout, err := time.ParseDuration(timeoutDuration)
	if err != nil {
		return errors.Wrapf(err, "Unable to parse timeout duration")
	}

	startTS := time.Now()
	for {
		statusResp, err := client.GetClusterStatus(ctx, &env_pb.GetClusterStatusReq{
			ClusterId: clusterId,
		})
		if err != nil {
			return errors.Wrapf(err, "Unable to read runtime link status for %s", clusterName)
		}

		if statusResp.LastHeartbeatTimestamp != nil {
			// consider a heartbeat within 10m as successfully linked
			healthyTS := time.Now().Add(-time.Minute * 10)
			if statusResp.LastHeartbeatTimestamp.AsTime().After(healthyTS) {
				return nil
			}
		}

		if time.Since(startTS) > timeout {
			return errors.Errorf("Timeout waiting for runtime link status, timeout: %s", timeoutDuration)
		}

		time.Sleep(time.Second * 1)
	}
}
