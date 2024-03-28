import base64
import json


def get_tenant_id_from_jwt(token: str) -> str:
    claims = extract_claims_from_jwt(token)

    return claims.get("sub")


def get_addresses_from_jwt(token: str) -> (str, str):
    claims = extract_claims_from_jwt(token)

    return claims.get("server_url"), claims.get("grpc_broadcast_address")


def extract_claims_from_jwt(token: str):
    parts = token.split(".")
    if len(parts) != 3:
        raise ValueError("Invalid token format")

    claims_part = parts[1]
    claims_part += "=" * ((4 - len(claims_part) % 4) % 4)  # Padding for base64 decoding
    claims_data = base64.urlsafe_b64decode(claims_part)
    claims = json.loads(claims_data)

    return claims
