import { atom, useAtom } from "jotai"

import { MailItems, mails } from "@/app/examples/mail/data"

type Config = {
  selected: MailItems["id"] | null
}

const configAtom = atom<Config>({
  selected: mails[0].id,
})

export function useMail() {
  return useAtom(configAtom)
}
