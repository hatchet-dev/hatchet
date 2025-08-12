import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    'package main\n\nimport (\n\t"fmt"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:"message"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tch := cmdutils.InterruptChan()\n\tcleanup, err := run()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("cleanup() error = %v", err))\n\t}\n}\n',
  source: 'out/go/z_v0/assignment-affinity/main.go',
  blocks: {},
  highlights: {},
};

export default snippet;
