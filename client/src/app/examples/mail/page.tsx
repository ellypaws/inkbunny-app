import { Mail } from "@/app/examples/mail/components/mail"
import { accounts } from "@/app/examples/mail/data"
import {useEffect, useState} from "react";

export default function MailPage() {
  const layout = { value: "[]" }
  const collapsed = { value: "[]" }

    const [mails, setMails] = useState([])
    const [loading, setLoading] = useState(false)

    useEffect(() => {
        // does things
        setLoading(true)
        fetch('/api/inkbunny/search?sid=guest&output=mail&temp=no')
            .then(r => r.json())
            .then(data => {
                console.log('GetMailResponse', data)
                setMails(data)
            }).finally(() => setLoading(false))
    }, [])

  const defaultLayout = layout ? JSON.parse(layout.value) : undefined
  const defaultCollapsed = collapsed ? JSON.parse(collapsed.value) : undefined

  return (
    <>
      <div className="md:hidden">
        <img
          src="/examples/mail-dark.png"
          width={1280}
          height={727}
          alt="Mail"
          className="hidden dark:block"
        />
        <img
          src="/examples/mail-light.png"
          width={1280}
          height={727}
          alt="Mail"
          className="block dark:hidden"
        />
      </div>
      <div className="">
        <Mail
          accounts={accounts}
          mails={mails}
          defaultLayout={defaultLayout}
          defaultCollapsed={defaultCollapsed}
          navCollapsedSize={4}
          loading={loading}
        />
      </div>
    </>
  )
}
