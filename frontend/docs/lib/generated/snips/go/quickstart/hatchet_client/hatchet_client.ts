import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package hatchet_client\n\nimport (\n\tv1 \"github.com/hatchet-dev/hatchet/pkg/v1\"\n\t\"github.com/joho/godotenv\"\n)\n\nfunc HatchetClient() (v1.HatchetClient, error) {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn v1.NewHatchetClient()\n}\n",
  "source": "out/go/quickstart/hatchet_client/hatchet_client.go",
  "blocks": {},
  "highlights": {}
};

export default snippet;
