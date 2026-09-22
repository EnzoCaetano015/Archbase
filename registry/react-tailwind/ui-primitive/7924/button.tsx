import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"
import { LoaderCircle } from "lucide-react"
import {
    forwardRef,
    type ButtonHTMLAttributes,
    type ElementRef,
} from "react"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
    "inline-flex items-center justify-center rounded-md font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50",
    {
        variants: {
            variant: {
                default: "bg-primary text-primary-foreground hover:bg-primary/90",
                outline: "border border-border bg-background hover:bg-muted",
            },
            size: {
                default: "h-10 px-4 py-2",
                icon: "size-10",
            },
        },
        defaultVariants: {
            variant: "default",
            size: "default",
        },
    },
)

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> &
    VariantProps<typeof buttonVariants> & {
        asChild?: boolean
        loading?: boolean
    }

export const Button = forwardRef<ElementRef<"button">, ButtonProps>(
    (
        {
            asChild = false,
            className,
            children,
            disabled,
            loading = false,
            size,
            variant,
            ...props
        },
        ref,
    ) => {
        const Component = asChild ? Slot : "button"

        return (
            <Component
                {...props}
                ref={ref}
                disabled={disabled || loading}
                aria-busy={loading || undefined}
                className={cn(buttonVariants({ variant, size }), className)}
            >
                {loading ? <LoaderCircle aria-hidden="true" className="animate-spin" /> : null}
                {children}
            </Component>
        )
    },
)

Button.displayName = "Button"
