import { Button } from "@/registry/new-york/ui/button"

export default function ButtonAsChild() {
  return (
    <Button asChild>
      <a href="/login">Login</a>
    </Button>
  )
}
