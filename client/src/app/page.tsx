import MailPage from "@/app/examples/mail/page"

export default function IndexPage() {
  return (
    <div className="container relative">
      {/*<Header/>*/}
      {/*<ExamplesNav className="[&>a:first-child]:text-primary" />*/}
      <section className="md:block">
        <div className="rounded-lg border max-h-[1000px]">
          <MailPage/>
        </div>
      </section>
    </div>
  )
}
