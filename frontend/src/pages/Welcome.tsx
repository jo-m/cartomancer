import { useAppConfig } from "../api/client"
import { useSession } from "../context/SessionContext"
import useDocumentTitle from "../hooks/useDocumentTitle"
import PageContainer from "../components/ui/PageContainer"
import Button from "../components/ui/Button"
import SvgIcon from "../assets/SvgIcon"
import ornamentDividerSvg from "../assets/ornament-divider.svg?raw"
import cardCornerSvg from "../assets/card-corner.svg?raw"
import scrollSvg from "../assets/scroll.svg?raw"
import crystalBallSvg from "../assets/crystalball.svg?raw"
import compassSvg from "../assets/compass.svg?raw"

interface Feature {
  title: string
  icon: string
  blurb: string
}

const features: Feature[] = [
  {
    title: "The Library",
    icon: scrollSvg,
    blurb:
      "Keep your GPX and FIT routes in one place. Tag them, search them, filter by sport, distance or elevation. Share with the world or keep them private.",
  },
  {
    title: "The Oracle",
    icon: crystalBallSvg,
    blurb:
      "Read wind, rain and temperature for the very hour you set off. Plan around the weather instead of being surprised by it.",
  },
  {
    title: "The Compass",
    icon: compassSvg,
    blurb:
      "Surface kindred routes nearby, by area, distance or shape. Discover paths you didn't know were there.",
  },
]

/** Welcome landing page shown to all visitors at the root route. */
export default function Welcome() {
  useDocumentTitle("")
  const { data: appConfig } = useAppConfig()
  const { user } = useSession()

  return (
    <PageContainer className="py-16">
      <header className="text-center">
        <h1 className="mb-4 text-4xl font-semibold tracking-wide text-text">
          {appConfig?.instanceName}
        </h1>
        <p className="mb-8 text-lg text-text-secondary">
          The gpx track library with a touch of magic.
        </p>
        <div className="flex flex-wrap justify-center gap-4">
          {user ? (
            <>
              <Button to="/tracks" variant="primary">
                Public tracks
              </Button>
              <Button to="/account/tracks" variant="secondary">
                My tracks
              </Button>
            </>
          ) : (
            <>
              <Button to="/tracks" variant="primary">
                Browse tracks
              </Button>
              <Button to="/login" variant="secondary">
                Log in
              </Button>
              {appConfig?.registrationEnabled && (
                <Button to="/register" variant="secondary">
                  Create account
                </Button>
              )}
            </>
          )}
        </div>
      </header>

      <div
        className="mt-16 mb-12 flex justify-center text-border"
        aria-hidden="true"
      >
        <SvgIcon svg={ornamentDividerSvg} className="h-2.5 w-40" />
      </div>

      <section
        aria-label="Features"
        className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3"
      >
        {features.map((feature) => (
          <article
            key={feature.title}
            className="relative rounded-lg border border-border bg-panel px-6 pb-7 pt-10 text-center"
          >
            <SvgIcon
              svg={cardCornerSvg}
              className="absolute -left-0.5 -top-0.5 h-5 w-5 text-border"
            />
            <SvgIcon
              svg={cardCornerSvg}
              className="absolute -right-0.5 -top-0.5 h-5 w-5 -scale-x-100 text-border"
            />
            <SvgIcon
              svg={cardCornerSvg}
              className="absolute -bottom-0.5 -left-0.5 h-5 w-5 -scale-y-100 text-border"
            />
            <SvgIcon
              svg={cardCornerSvg}
              className="absolute -bottom-0.5 -right-0.5 h-5 w-5 -scale-x-100 -scale-y-100 text-border"
            />

            <div className="mx-auto mb-5 inline-flex h-16 w-16 items-center justify-center rounded-full border border-border bg-surface text-primary">
              <SvgIcon svg={feature.icon} className="h-9 w-9" />
            </div>
            <h2 className="mb-2 text-2xl tracking-wide text-text">
              {feature.title}
            </h2>
            <p className="text-sm leading-relaxed text-text-secondary">
              {feature.blurb}
            </p>
          </article>
        ))}
      </section>

      <div className="mt-12 flex justify-center text-border" aria-hidden="true">
        <SvgIcon svg={ornamentDividerSvg} className="h-2.5 w-40" />
      </div>
    </PageContainer>
  )
}
