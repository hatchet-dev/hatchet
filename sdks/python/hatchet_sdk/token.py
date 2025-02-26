import base64

from pydantic import BaseModel


class Claims(BaseModel):
    sub: str
    server_url: str
    grpc_broadcast_address: str


def get_tenant_id_from_jwt(token: str) -> str:
    return extract_claims_from_jwt(token).sub


def get_addresses_from_jwt(token: str) -> tuple[str, str]:
    claims = extract_claims_from_jwt(token)

    return claims.server_url, claims.grpc_broadcast_address


def extract_claims_from_jwt(token: str) -> Claims:
    parts = token.split(".")
    if len(parts) != 3:
        raise ValueError("Invalid token format")

    claims_part = parts[1]
    claims_part += "=" * ((4 - len(claims_part) % 4) % 4)  # Padding for base64 decoding
    claims_data = base64.urlsafe_b64decode(claims_part)

    return Claims.model_validate_json(claims_data)
