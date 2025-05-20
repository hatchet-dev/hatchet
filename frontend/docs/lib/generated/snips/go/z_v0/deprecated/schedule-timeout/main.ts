import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"time\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client\"\n\t\"github.com/joho/godotenv\"\n)\n\ntype sampleEvent struct{}\n\ntype timeoutInput struct{}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New(\n\t\tclient.InitWorkflows(),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevent := sampleEvent{}\n\n\t// push an event\n\terr = client.Event().Push(\n\t\tcontext.Background(),\n\t\t\"user:create\",\n\t\tevent,\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\ttime.Sleep(35 * time.Second)\n\n\tfmt.Println(\"step should have timed out\")\n}\n",
  "source": "out/go/z_v0/deprecated/schedule-timeout/main.go",
  "blocks": {},
  "highlights": {}
};

export default snippet;
