def get_metadata(token: str) -> tuple[tuple[str, str], ...]:
    """
    Get metadata for gRPC calls including authorization and compression headers.
    
    The grpc-accept-encoding header must be set per-call because Python gRPC
    channel options don't override per-call metadata. This ensures the server
    knows the client accepts gzip compression.
    """
    return (
        ("authorization", "bearer " + token),
        ("grpc-accept-encoding", "gzip,identity"),
    )
