import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\t\'context\'\n\t\'fmt\'\n\t\'time\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client\'\n\t\'github.com/hatchet-dev/hatchet/pkg/cmdutils\'\n\t\'github.com/joho/godotenv\'\n)\n\ntype Event struct {\n\tID        uint64    `json:\'id\'`\n\tCreatedAt time.Time `json:\'created_at\'`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:\'message\'`\n}\n\nfunc StepOne(ctx context.Context, input *Event) (result *stepOneOutput, err error) {\n\tfmt.Println(input.ID, \'delay\', time.Since(input.CreatedAt))\n\n\treturn &stepOneOutput{\n\t\tMessage: \'This ran at: \' + time.Now().Format(time.RubyDate),\n\t}, nil\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tvar id uint64\n\tgo func() {\n\t\tfor {\n\t\t\tselect {\n\t\t\tcase <-time.After(5 * time.Second):\n\t\t\t\tfor i := 0; i < 100; i++ {\n\t\t\t\t\tid++\n\n\t\t\t\t\tev := Event{CreatedAt: time.Now(), ID: id}\n\t\t\t\t\tfmt.Println(\'pushed event\', ev.ID)\n\t\t\t\t\terr = client.Event().Push(interruptCtx, \'test:event\', ev)\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\tpanic(err)\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\tcase <-interruptCtx.Done():\n\t\t\t\treturn\n\t\t\t}\n\t\t}\n\t}()\n\n\tfor {\n\t\tselect {\n\t\tcase <-interruptCtx.Done():\n\t\t\treturn\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n}\n',
  'source': 'out/go/z_v0/loadtest/emitter/main.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
