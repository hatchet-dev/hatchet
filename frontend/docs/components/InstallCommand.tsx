import { useLanguage } from "@/context/LanguageContext";
import { Tabs, Code } from "nextra/components";
import UniversalTabs from "@/components/UniversalTabs";
import { CodeBlock } from "./code/CodeBlock";

export default function InstallCommand({
    installOnly,
    withDevDependencies
}: {
    installOnly?: boolean,
    withDevDependencies?: boolean
}) {
    const {
        selectedLanguage,
    } = useLanguage();
    
    if (selectedLanguage === "Typescript") {
        return (
            <UniversalTabs items={["npm", "pnpm", "yarn"]} optionKey="packageManager">
                <Tabs.Tab title="npm">
                    <CodeBlock source={{
                        language: "bash",
                        raw: `npm ${installOnly ? "install" : "i @hatchet-dev/typescript-sdk"} ${withDevDependencies ? "&& npm i ts-node dotenv typescript --save-dev" : ""}`,
                    }} />
                </Tabs.Tab>
                <Tabs.Tab title="pnpm">
                    <CodeBlock source={{
                        language: "bash",
                        raw: `pnpm ${installOnly ? "add" : "i @hatchet-dev/typescript-sdk"} ${withDevDependencies ? "&& pnpm i ts-node dotenv typescript --save-dev" : ""}`,
                    }} />
                </Tabs.Tab>
                <Tabs.Tab title="yarn">
                    <CodeBlock source={{
                        language: "bash",
                        raw: `yarn ${installOnly ? "add" : "i @hatchet-dev/typescript-sdk"} ${withDevDependencies ? "&& yarn add ts-node dotenv typescript --dev" : ""}`,
                    }} />
                </Tabs.Tab>
            </UniversalTabs>    
        );
    } else if (selectedLanguage === "Python") {
        return withDevDependencies ? (
            <UniversalTabs items={["pip", "poetry"]} optionKey="packageManager">
                <Tabs.Tab title="pip">
                    <CodeBlock source={{
                        language: "bash",
                        raw: `pip install hatchet-sdk`,
                    }} />
                </Tabs.Tab>
                <Tabs.Tab title="poetry">
                    <CodeBlock source={{
                        language: "bash",
                        raw: `poetry add hatchet-sdk`,
                    }} />
                </Tabs.Tab>
            </UniversalTabs>
        ) : (
            <UniversalTabs items={["poetry", "pip"]} optionKey="packageManager">
                <Tabs.Tab title="poetry">
                    <h6>Create a virtual environment</h6>
                    <CodeBlock source={{
                        language: "bash",
                        raw: `poetry shell`,
                    }} />
                    <h6>Initialize the project and install the Hatchet SDK</h6>
                    <CodeBlock source={{
                        language: "bash",
                        raw: `poetry init --dependency hatchet-sdk`,
                    }} />
                </Tabs.Tab>
                <Tabs.Tab title="pip">
                    <h6>Create a virtual environment</h6>
                    <CodeBlock source={{
                        language: "bash",
                        raw: `python -m venv venv
source venv/bin/activate  # On Unix/macOS`,
                    }} />
                    <h6>Install the Hatchet SDK</h6>
                    <CodeBlock source={{
                        language: "bash",
                        raw: `pip install hatchet-sdk`,
                    }} />
                    <h6>(optional) save the dependencies to a requirements.txt file</h6>
                    <CodeBlock source={{
                        language: "bash",
                        raw: `pip freeze > requirements.txt`,
                    }} />
                </Tabs.Tab>
            </UniversalTabs>
        );
    } else if (selectedLanguage === "Go") {
        return (
            <CodeBlock source={{
                language: "bash",
                raw: `go mod tidy`,
            }} />
        );
    }
    return (
        <div>
            <h1>Select a language {selectedLanguage}</h1>
        </div>
    );
}