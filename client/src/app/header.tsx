import {PageActions, PageHeader, PageHeaderDescription, PageHeaderHeading} from "@/components/page-header.tsx";
import {siteConfig} from "@/config/site.ts";
import {cn} from "@/lib/utils.ts";
import {buttonVariants} from "@/registry/new-york/ui/button.tsx";
import {Icons} from "@/components/icons.tsx";

export default function Header() {
    return <PageHeader>
        {/*<Announcement />*/}
        <PageHeaderHeading>{siteConfig.name}</PageHeaderHeading>
        <PageHeaderDescription>
            {siteConfig.description}
        </PageHeaderDescription>
        <PageActions>
            <a href="/docs" className={cn(buttonVariants())}>
                Get Started
            </a>
            <a
                target="_blank"
                rel="noreferrer"
                href={siteConfig.links.github}
                className={cn(buttonVariants({variant: "outline"}))}
            >
                <Icons.gitHub className="mr-2 h-4 w-4"/>
                GitHub
            </a>
        </PageActions>
    </PageHeader>;
}