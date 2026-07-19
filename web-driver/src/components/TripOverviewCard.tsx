import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card"
import { ScrollArea } from "./ui/scroll-area"

interface TripOverviewCardProps {
  title: string
  description: string
  children?: React.ReactNode
  className?: string
}

export const TripOverviewCard = ({ title, description, children, className }: TripOverviewCardProps) => {
  return (
    <Card className={`w-full border border-emerald-900/10 shadow-sm ${className ?? ""}`}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        <ScrollArea>
          {children}
        </ScrollArea>
      </CardContent>
    </Card>
  )
}
