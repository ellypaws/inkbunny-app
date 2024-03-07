import { atom, useAtom } from "jotai"

import { MailItem, mails } from "@/app/examples/mail/data"

type Config = {
  selected: MailItem["id"] | null
}

const configAtom = atom<Config>({
  selected: mails[0].id,
})

export function useMail() {
  return useAtom(configAtom)
}
