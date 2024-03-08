import { Mail } from "@/app/examples/mail/components/mail"
import { accounts } from "@/app/examples/mail/data"
import {useEffect, useState} from "react";

export default function MailPage() {
  const layout = { value: "[]" }
  const collapsed = { value: "[]" }

    const [mails, setMails] = useState([])
    const [loading, setLoading] = useState(false)

    useEffect(() => {
        setLoading(true)
        const urlParams = new URLSearchParams(window.location.search);
        const temp = urlParams.get('temp') || 'no'

        fetch(`/api/inkbunny/search?sid=guest&output=mail&temp=${temp}`)
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
