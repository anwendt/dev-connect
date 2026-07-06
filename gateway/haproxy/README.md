# HAProxy Gateway

This directory is reserved for future HAProxy-specific assets.

The approved design uses one HAProxy TCP listener on container port `2222`, exposed by a Kubernetes Service on port `22`, with one backend target per gateway Deployment.
