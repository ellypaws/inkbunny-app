import MailPage from "@/app/examples/mail/page"
import TaskPage from "@/app/examples/tasks/page"

import { useRoutes, Link, useQueryParams } from 'raviger'

const routes = {
    '/': () => <MailPage />,
    '/tasks': () => <TaskPage />,
}

export default function IndexPage() {
  let route = useRoutes(routes)
  return (
    <div className="container relative">
      {/*<Header/>*/}
      {/*<ExamplesNav className="[&>a:first-child]:text-primary" />*/}
      <section className="md:block">
        <div className="rounded-lg border max-h-[1000px]">
            {route}
        </div>
      </section>
    </div>
  )
}
