import { ApplicationNavigation } from "@/shared/ui/application-navigation"
import { ApplicationShell } from "@/shared/ui/application-shell"

type PlatformLayoutProps = Readonly<{
  children: React.ReactNode
}>

export default function PlatformLayout({ children }: PlatformLayoutProps) {
  return (
    <>
      <ApplicationNavigation />
      <ApplicationShell>{children}</ApplicationShell>
    </>
  )
}
