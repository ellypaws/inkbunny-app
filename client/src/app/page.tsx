

import { siteConfig } from "@/config/site"
import { cn } from "@/lib/utils"
import { Icons } from "@/components/icons"
import {
  PageActions,
  PageHeader,
  PageHeaderDescription,
  PageHeaderHeading,
} from "@/components/page-header"
import { buttonVariants } from "@/registry/new-york/ui/button"
import MailPage from "@/app/examples/mail/page"

export default function IndexPage() {
  return (
    <div className="container relative">
      <PageHeader>
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
            className={cn(buttonVariants({ variant: "outline" }))}
          >
            <Icons.gitHub className="mr-2 h-4 w-4" />
            GitHub
          </a>
        </PageActions>
      </PageHeader>
      {/*<ExamplesNav className="[&>a:first-child]:text-primary" />*/}
      <section className="md:block">
        <div className="rounded-lg border bg-background shadow-lg">
          <MailPage />
        </div>
      </section>
    </div>
  )
}
