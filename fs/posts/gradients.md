---
is_published: true

title: Improving gradients in TailwindCSS
# optional
subtitle: A deep dive into how we can improve gradient utility classes in TailwindCSS

# optional
landing_pitch: An unnecessarily deep dive into improving a teeny subset of TailwindCSS's utility classes.

# optional
meta_description: Let's learn a bit about gradients, color interpolation, W3C syntax, and the Tailwind plugin API to improve Tailwind's set of gradient utility class!

upload_date: 20 April 2024
modified_date: 20 April 2024
reading_time: 40

use_dynamic_og: false
og_alt: Three different circular gradients overlaid atop of a bunch of TailwindCSS classes. The title of the banner reads 'Improving gradients in TailwindCSS.'
---

For those unfamiliar, [TailwindCSS](https://tailwindcss.com/) is a CSS framework that allows developers to write using _utility-first classes._ A developer may typically write a card like `<div class="calendar">...</div>`. The `.calendar` class would then have a long list of CSS properties that style the div. With TailwindCSS, that div may look something like:

```html
<div class="bg-zinc-100 p-4 rounded-lg flex flex-col gap-2 ..."></div>
```

Each of those classes usually represents a _single_ CSS property. `bg-zinc-100`, for example, sets the `background-color` property to a light, muted blue. `p-4` sets the padding of the element to `1rem`. Tailwind gives us a neutral set of classes, which we can use to craft components. I personally _love_ Tailwind—nearly any web project beyond a small prototype uses Tailwind.

Tailwind is definitely polarizing. We don’t need to swap between HTML and CSS files, however, it’s easy to fill elements with a ridiculous number of classes. Like, One of the elements on this blog literally looks like this:

```html
<main class="max-w-none w-full px-5 4k:px-16 prose prose-zinc dark:prose-invert
  prose-code:before:content-none prose-code:after:content-none
  dark:prose-h1:text-zinc-200 dark:prose-h2:text-zinc-300 dark:prose-h3:text-zinc-300 dark:prose-p:text-zinc-300 prose-a:text-inherit dark:prose-a:text-zinc-200
  prose-h2:mt-8 prose-h2:mb-6 last-of-type:prose-p:mb-0 prose-ul:mb-3 prose-ol:mb-3
  first-of-type:prose-th:rounded-l-md last-of-type:prose-th:rounded-r-md
  prose-th:py-2 has-[th]:prose-tr:bg-zinc-100 dark:has-[th]:prose-tr:bg-zinc-900 prose-thead:sticky prose-thead:top-0">
  <!-- ... -->
</main>
```

Yeah… holy smokes I can understand why a pretty nasty element like that can be pretty cumbersome and off-putting to someone unfamiliar with the syntax. By being utility-first, Tailwind confronts the standard way we write CSS. It's different, no doubt, but I think it's a little easy to conflate _different_ or _verbose_ with _bad_ and _unmaintainable_.

I personally find Tailwind to fall a little short in a few places—and I think the way that Tailwind handles gradients is in sore need of a revisit. So, in this post, we’ll do exactly that! We’ll work with the Tailwind plugin API to improve and extend the existing set of utility classes that Tailwind provides to deal with gradients.

By the end of this _very_ lengthy post, we will have:

- A good understanding of how Tailwind handles gradients, and where it falls short
- Knowledge of how W3C defines a gradient, and how to read W3C syntax
- An introduction to Tailwind plugins by generalizing the `linear-gradient()` direction utility classes
- A review of color interpolation methods, and how we can implement a set of utility classes to handle them _while respective progressive enhancement_
- A deep understanding of how the `linear-gradient()`, `radial-gradient()`, and `conic-gradient()` functions all differ from one another, and how we can further extend our gradient plugin by adding support for the latter two.

It’s a long post, so let’s jump in! ᕕ( ᐛ )ᕗ

## How Tailwind handles gradient

Currently, Tailwind CSS supports a pretty limited number of solutions when it comes to creating utility-based gradients. Gradients are defined with a series of classes in the `background-image` utility:

- `bg-gradient-to-*`: this adds the `background-image` property to our element, and—based on the class— specifies the direction of the gradient.
- `from-*`: Our first gradient stop. This can be written to specify a color as `from-<color>`. We can optionally add a `from-<percentage>` class, which tells us where along the gradient the color stop is located.
- `via-*`: The middle gradient stop. This utility has the same forms as `from-*`.
- `to-*`: The final gradient stop.

A Tailwind gradient assumes the `from-*`, `via-*`, and `to-*` stop colors to be `transparent`, so we only _need_ to define one color stop. Defining a gradient in Tailwind utility classes can be as complex as:

```html
<div class="bg-gradient-to-<direction>
  from-<color> from-<percentage>
  via-<color> via-<percentage>
  to-<color> to-<percentage>">
</div>
```

However, this misses out on a lot of functionality. Let’s take a peek at how the `linear-gradient()` function is defined.

### How are gradients _really_ defined?

The [W3C](https://www.w3.org/) is the body that writes the **standards** for the web. These standards tell browsers and developers the expectations for everything that makes the web what it is: HTML, CSS, Web APIs, accessibility, internationalization, etc.

When it comes to CSS, the W3C has a standard way of writing **syntax**. Syntax tells us how things in CSS can be written validly. For example, the syntax of the `<color>` type is:

```html
<color> = <color-base> | currentColor | <system-color>
```

The bar (`|`) combinator tells us that a valid color can be either `<color-base>` _or_ `currentColor` _or_ a `<system-color>`. A valid `<color>` could be `hsl(35deg 100% 50%)` because that’s a valid `<color-base>`, but values like `#a`, `woofdog`, or `superDeepPurple` are _not_ valid.

The `<linear-gradient()>` type tells us how we can write a valid `linear-gradient()` function:

```html
<linear-gradient()> = linear-gradient( <linear-gradient-syntax> )
```

And the `<linear-gradient-syntax>` type tells us the valid arrangement of the `linear-gradient()` arguments:

```html
<linear-gradient-syntax> =
  [ [ <angle> | to <side-or-corner> ] || <color-interpolation-method> ]?,
  <color-stop-list>
```

Looks a bit complex! We can break this down piece by piece though. We can first note the **grouping brackets** (`[ ]`) that surround the first chunk, followed by the **question mark** (`?`). This tells us that everything inside the brackets (`[ <angle> | to <side-or-corner> ] || <color-interpolation-method>`) is optional.

Inside these first set of brackets we have the following:

```html
[ <angle> | to <side-or-corner> ] || <color-interpolation-method>
```

The **double bar** (`||`) tells us that we can have _one or more_ of the options on either side of it. We can think of it as the `option` combinator: we can pick multiple components if we want and add them into the mix. We can include just one portion of this syntax, like `[ <angle> | to <side-or-corner> ]` or `<color-interpolation-method>`, but we can also include the whole thing. An example `linear-gradient()` function might be:

```css
background: linear-gradient(35deg in oklab, /* ... */);
```

The `35deg` satisfies the `[ <angle> | to <side-or-corner> ]` syntax, and the `in oklab` satisfies the `<color-interpolation-method>` syntax.
On the left side of this double bar (`||`) is another set of brackets that enclose `<angle> | to <side-or-corner>`. Here, the **single bar** (`|`) combinator tells us that we choose _only one_ of the components within its scope. In this case, we can choose _either_ an `<angle>` (like `35deg`) or a `to <side-or-corner>` component (like `to top left`).

There’s a lot to it, but reading the syntax isn’t too bad, I promise!

Ultimately, there are a few general components to focus on:

1. The **direction** of the **gradient line** (`[ <angle> | to <side-or-corner> ]`): We can think of a linear gradient as **interpolating** (changing colors) over a line—the direction of the linear gradient tells us the rotation of that gradient line (0deg points upwards).

   This component defaults to `to bottom`, meaning the gradient starts at the top of the element, and interpolates towards the bottom of the element.

2. The **color interpolation method** (`<color-interpolation-method>`): This essentially tells us how the gradient interpolates between two colors. We’ll cover this much deeper later!

   [My understanding](https://www.w3.org/TR/css-color-4/#interpolation-space) is that this component defaults to `in oklab`, but it seems like some browsers actually use `in srgb` (don’t worry it isn’t super important to know what this means right now).

3. The **color stops** (`<color-stop-list>`): This is a list of colors and percentages, which defines the colors the gradient interpolates between, and over what portion of the gradient we interpolate between two colors. The `<color-stop-list>` syntax is defined as:

   ```html
   <color-stop-list> = <linear-color-stop>, [ <linear-color-hint>? , <linear-color-stop> ]#
   ```

   We won’t dive too deep into this, but in general, this syntax allows us to define as many color stops as we want. The `<linear-color-stop>` type allows us to specify a color and up to two `<length-percentage>` types (these are either a length, like `50px`, or a percentage, like `10%`). These two `<length-percentage>` types specify the start and end of where the specified color is _solid_ (i.e. where it doesn’t change at all).

   After we define one color stop, we can define zero or more `[ <linear-color-hint>? , <linear-color-stop> ]` groups. A `<linear-color-hint>` tells us where the interpolation between the previous color hits its “halfway point”. Take the following `linear-gradient()` function:

   ```css
   background: linear-gradient(red, 20px, blue);
   ```

   This tells us that the gradient starts at red at `0px`, interpolates halfway towards blue by `20px`, and then interpolates to blue until the end of the gradient.

   The group also tells us that we specify another `<linear-color-stop>`, which takes the same form as the first. This syntax allows us to build out an indefinitely long list of color stops. I don’t want to dive too deep into this, because it’s surprisingly complex for a tiny part of syntax within a larger whole.

### Where does Tailwind Fall Short?

So, we know a bit more about every part of the `linear-gradient()` function. But where does Tailwind fall short?

In Tailwind, we first define a gradient with one of the `bg-gradient-to-*` utility classes. the asterisk is just a stand-in, and this stand-in can be one of eight directions: `t`, `tr`, `r`, `br`, `b`, `bl`, `l`, and `tl`. These **shorthand** properties correspond to one of the valid `<angle-or-corner>` data types.

After we define a gradient, we can use the `from-*`, `via-*`, and `to-*` utility classes to define the initial, middle, and end color stops (and their respective positions).

For example, we may use the following Tailwind utility classes to define a gradient:

```html
<div class="bg-gradient-to-r from-red-500 from-10% to-blue-500 to-90%"></div>
```

This is syntactically analogous to the following CSS:

```css
background-image: linear-gradient(
  to right,
  rgb(239 68 68) 10%,
  rgb(59 130 246) 90%
);
```

But this is essentially as complex as we can get! We _could_ define a middle point, but that’s pretty much it! We’re missing a bit here:

1. **The direction is restricted**: we can choose an `<angle-or-corner>` (like `bg-gradient-to-t`), but can’t use any `<angle>` type (for example, we can’t do anything like `bg-gradient-to-[35deg]`).
2. **We can’t specify any interpolation method**: Tailwind’s set of utility classes for gradients doesn’t give us any access to change how the gradient interpolates between two colors. For 95% of cases this is no big deal, but we _should_ have access to this, right? There are small—but noticeable—differences between an interpolation in sRGB and an interpolation in LAB!
3. **We can’t change the gradient we’re using**: beyond `linear-gradient()`, CSS also has `radial-gradient()` and `conic-gradient()` functions. We’ll dig into how these gradient functions differ later in this post—the thing to note here is that Tailwind has no way to access these (outside of an arbitrary property class like `[background-image:conic-gradient(...)]`, which is pretty cumbersome).
4. **We can’t specify more than three color stops**: The syntax for `<color-stop-list>` allows us to define as many color stops as we want, however, we’re restricted to only three.

We won’t cover this last missing detail in this post, however. I generally think three color stops are fine for nearly any gradient, and the syntax of `<color-stop-list>` is pretty wacky. How would Tailwind theoretically even handle the potentiality of _infinitely many color stops_???

The first three issues are fair game though, and adding support for them with Tailwind-like syntax is the main focus of this post!

## Improving gradient directions

The preexisting `bg-gradient-to-*` utility classes only support the `<side-or-corner>` data type, so we only direct our gradient in one of these eight directions. This is fine for most purposes, however, there’s nothing wrong with adding support for a more granular set of classes that support the `<angle>` data type.

Tailwind has some default rotation mappings through the `rotate` theme property. For example, `12` corresponds to `12deg`. This property would then be used to generate a set of utility classes, like `rotate-*`. When we then add, say, `rotate-12` to an element, we get the generated CSS declaration `rotate: 12deg`.

Tailwind’s `matchUtilities()` plugin function allows us to register a bunch of utility classes at once, by mapping keys and values to utility class names and generated properties, respectively:

```jsx
matchUtilities(
  {
    [`bg-gradient-to`]: (val) => {
      return {
        "background-image": `linear-gradient(${val} var(--tw-color-interpolation-method, ), var(--tw-gradient-stops,))`,
      };
    },
  },
  {
    values: theme("rotate"),
  }
);
```

With this improvement, we automatically gain a bunch of new utility classes for every key/value pair in the `rotate` theme. A dev can even extend or modify this theme property…

```json
export default {
  theme: {
		extend: {
			rotate: {
				77: "77deg",
			}
		}
	}
}
```

…and we’ll automatically generate a corresponding `bg-gradient-to-77` class! A side benefit of this is that we can use the arbitrary value bracket syntax to specify gradients with any angle we want:

```html
<!-- Tailwind-provided named direction class -->
<div class="bg-gradient-to-r"></div>

<!-- angle class -->
<div class="bg-gradient-to-75"></div>

<!-- angle class: arbitrary value-->
<div class="bg-gradient-to-[37deg]"></div>
```

Just for completeness, here’s the full list of the linear gradient angle utility classes and their generated class properties:

| Tailwind class       | Generated declarations                                                 |
| -------------------- | ---------------------------------------------------------------------- |
| `bg-gradient-to-0`   | `background-image: linear-gradient(0deg, var(--tw-gradient-stops));`   |
| `bg-gradient-to-1`   | `background-image: linear-gradient(1deg, var(--tw-gradient-stops));`   |
| `bg-gradient-to-2`   | `background-image: linear-gradient(2deg, var(--tw-gradient-stops));`   |
| `bg-gradient-to-3`   | `background-image: linear-gradient(3deg, var(--tw-gradient-stops));`   |
| `bg-gradient-to-6`   | `background-image: linear-gradient(6deg, var(--tw-gradient-stops));`   |
| `bg-gradient-to-12`  | `background-image: linear-gradient(12deg, var(--tw-gradient-stops));`  |
| `bg-gradient-to-45`  | `background-image: linear-gradient(45deg, var(--tw-gradient-stops));`  |
| `bg-gradient-to-90`  | `background-image: linear-gradient(90deg, var(--tw-gradient-stops));`  |
| `bg-gradient-to-180` | `background-image: linear-gradient(180deg, var(--tw-gradient-stops));` |

## Implementing color space interpolation

The [`<color-interpolation-method>` data type](https://www.w3.org/TR/css-color-4/#interpolation-space) is a relatively new addition to gradient functions. With this new data type, we can specify the **color space** that the gradient will use to interpolate between colors.

### What are color spaces, even?

Color science is a _deep_ topic, so we won’t dive too deep into it here. A **color space** is essentially a method of arranging and describing these colors. For example, **sRGB** allows us to specify a color using three values—one for red, one for green, and one for blue: `color: rgb(98, 244, 230);`

Different color spaces give us different ways of accessing colors and have different pros and cons. sRGB, for example, isn’t **perceptually uniform**: changes in hue _appear_ to differ wildly in apparent lightness and saturation. Something like Oklab, on the other hand, is **perceptually uniform**: changes in hue appear to have the same lightness to them.

This is highly relevant when it comes to gradients since these are _all about_ moving around a color space! If we interpolate from one color to another in sRGB, we aren’t guaranteed a perceptually uniform result across the gradient (notice the muted colors in the middle):

But, a gradient that uses a perceptually uniform color space (like Oklab) doesn’t have this issue:

The difference is subtle. Notice the muted colors in the middle of the sRGB gradient and the more vibrant colors in the middle of the LAB gradient. Because LAB is perceptually uniform, the gradient appears to be more **consistent** in terms of lightness and saturation.

I'm not an expert on this at all. I develop more frequently than I design, so I don't exactly have that precise designer eye. What's worse is that I'm not even well versed enough in the intricacies of color gamuts and spaces to build a neat tool to visualize any of this! Luckily, other people have already done a better job at this than I could probably ever do.

Adam Argyle’s [High Definition CSS Color Guide](https://developer.chrome.com/docs/css-ui/high-definition-css-color-guide#color_interpolation) is the guide when it comes to color spaces and CSS (and is relevant to a lot of what we talk about in this section). I also highly recommend Eric Portis’ _[incredible_ write-up on color spaces](https://ericportis.com/posts/2024/okay-color-spaces/), which delves into a lot more into the science and history of color theory. For an amazing visual tool, Isaac Muse’s interactive [ColorAide color space viewer](https://facelessuser.github.io/coloraide/demos/3d_models.html) is unbeatable. It allows you to map different color gamuts to color spaces (including a ton of spaces not natively available on the web).

### How color spaces are specified in gradient functions

The syntax of `<color-interpolation-method>` is:

```html
in [ <rectangular-color-space> | <polar-color-space> <hue-interpolation-method>? ]
```

We have two options here. On one hand, we can choose a rectangular color space (which we access with three linear axes):

```html
<rectangular-color-space> = srgb | srgb-linear | lab | oklab | xyz
```

On the other hand, we can choose a polar color space (which we access with two linear axes and a “rotation”). If we choose a polar color space to interpolate our gradient with, we can optionally choose a hue interpolation method, which tells us how we want to rotate around the color space to interpolate from one color to another.

<!-- <PolarHueInterp/> -->

```html
<polar-color-space> = hsl | hwb | lch | oklch
<hue-interpolation-method> = [shorter | longer | increasing | decreasing] hue
>
```

We can use the handy tool below to see how a given color interpolation method differs from the one the browser ships with. Note that, as of writing this, Firefox _does not support this new gradient syntax_... this tool does progressively enhance so it should work automatically as soon as Firefox supports gradient interpolation methods

<!-- <InterpList /> -->

The differences are subtle, but they matter! A gradient interpolation through `srgb` [might look pretty different](http://tavmjong.free.fr/SVG/COLOR_INTERPOLATION/) (i.e. worse) when compared to one interpolated through `lab` or `oklch`. Tailwind currently ignores the `<color-interpolation-method>` entirely, so we don’t have a way to easily specify any color interpolation space. We can support this cool feature, but there are a few things we need to note:

- **We need to be able to target the default `bg-gradient-to-*` utility classes**. The direction classes (e.g. `bg-gradient-to-r`) that Tailwind provides by default aren’t defined in the same place as our custom, `<angle>`-based utility classes. If we don’t figure out how to target these default classes, we’ll end up unnecessarily splintering support for interpolation methods across Tailwind classes, which would be pretty nasty!
- **We shouldn’t force any interpolation method by default**. The W3C does specify `oklab` as the default color space interpolation, however this isn’t followed in practice. For example, a `linear-gradient()` function in Chrome will default to interpolation in `srgb` if no color interpolation method is provided.
- **We need to account for browsers that lack support**. Firefox, for example, currently doesn’t support the new `linear-gradient()` syntax, so we can’t use color interpolation methods. Firefox doesn’t have a graceful fallback either, so we _need_ a linear-gradient without any `<color-interpolation-method>`.

So, in the previous section, we registered a bunch of `<angle>`-based utility classes. To support color interpolation methods, we can modify this a little bit.

### Overriding Tailwind’s default classes

We could modify the function we wrote earlier to accept interpolation methods, however, we wouldn’t see this new functionality reflected in the default, 8-way `bg-gradient-to-*` functions that Tailwind provides by default, since we only target the `rotate` property.

To remedy this, we can define a _custom_ theme property. We’ll call it `gradientDirections`. In this property, we’ll define our 8 possible values with the `<angle-or-corner>` data type, along with the theme’s `rotate` property:

```jsx
export default {
  // ...
  theme: {
    gradientDirection: ({ theme }) => ({
      // <side-or-corner>
      t: "to top",
      tr: "to top right",
      r: "to right",
      // ...
      ...theme("rotate"),
    }),
  },
  // ...
};
```

Then, we can slightly modify our `matchUtilities()` function to use the `gradientDirection` theme property, rather than the `rotate` property:

```jsx
matchUtilities(
  {
    ["bg-gradient-to"]: (val) => {
      return {
        "background-image": `linear-gradient(${val} var(--tw-color-interpolation-method, ), var(--tw-gradient-stops,))`,
      };
    },
  },
  {
    values: theme("gradientDirection"),
  }
);
```

There’s a slight problem, though. This… works… however Tailwind’s directions gradient classes are actually defined under the `backgroundImage` property. Because we’re registering our custom gradient classes under a custom `gradientDirection` property, Tailwind can’t see that there are technically two class definitions:

- `bg-gradient-to-t`: where `t` is a property in `backgroundImage` (Tailwind’s default)
- `bg-gradient-to-t`: where `t` is a property in `gradientDirection` (Our custom)

Because this isn’t visible, Tailwind ends up generating two class definitions for directional gradient classes. Check out the hover preview for one of these directional gradient classes:

This isn’t _technically_ a problem, but DX is always a nice thing, and conflicting CSS rules can be unnecessarily confusing.

_Why not just extend the `backgroundImage` property with rotational values?_ It’s a good thought, however, if we dive into Tailwind’s source code, we can see that the gradient classes are defined as **static utilities**, which means they are meant to not be :

```jsx
for (let [value, direction] of [
  ["t", "top"],
  ["tr", "top right"],
  ["r", "right"],
  ["br", "bottom right"],
  ["b", "bottom"],
  ["bl", "bottom left"],
  ["l", "left"],
  ["tl", "top left"],
]) {
  staticUtility(`bg-gradient-to-${value}`, [
    [
      "background-image",
      `linear-gradient(to ${direction}, var(--tw-gradient-stops,))`,
    ],
  ]);
}
```

Another issue is that `backgroundImage` already handles some arbitrary values. Besides the directional gradient classes, `backgroundImage` also registers:

- `bg-none`: which sets the `background-image` property to `none` (pretty self-explanatory lol)
- `bg-[*]`: an arbitrary value class that allows us to set an element’s `background-image` property to a resource defined with something like a `url()` function.

Now, we can specify a data type to match an arbitrary value, so we don’t accidentally create arbitrary value classes with the wrong data type:

```jsx
matchUtilities(
  {
    circle: (val) => {
      return {
        width: val,
        height: val,
        "border-radius": "50%",
      };
    },
  },
  {
    type: "length", // only accept lengths: 5px, 4rem, etc.
  }
);
```

The showstopper here is that the `<angle>` data type is, for some reason, not listed here! We have no way to differentiate arbitrary values such that an `<angle>` maps to `bg-gradient-to-*` and everything else to `bg-*`.

Luckily, we can solve this annoying issue. Tailwind allows us to disable _core plugins_, which are essentially sections of the Tailwind library. `backgroundImage`, for example, is one such core plugin. We’ll disable that:

```jsx
export default {
  theme: {
    // ...
  },
  corePlugins: {
    backgroundImage: false,
  },
  // ...
};
```

This removes the `bg-none` and `bg-*` classes, so we’ll need to reregister those. We’re essentially going to take control over generating the `backgroundImage` classes. We’ll redefine _only_ `none` in the theme’s `backgroundImage` property:

```jsx
export default {
  theme: {
    gradientDirection: {
      // ...
    },
    backgroundImage: {
      none: "none",
    },
  },
  corePlugins: {
    backgroundImage: false,
  },
  // ...
};
```

Then, we’ll register this `backgroundImage` property as a set of dynamic utilities, so we can also get the arbitrary value support. We need to make sure we restrict the type of arbitrary values we accept here, since `bg-[*]` can _also_ represent an arbitrary background-color utility, like `bg-[#ff00ff]` (these arbitrary classes are defined under the `backgroundColor` plugin, so we don’t need to worry about losing `background-color` utility support when we disable the `backgroundImage` plugin).

```html
<!-- this should generate "background-color: #ff00ff;" -->
<div class="bg-[#ff00ff]"></div>

<!-- this should generate "background-image: url('/img/logo.svg');" -->
<div class="bg-[url('/img/logo.svg')]"></div>

<!-- this should generate "background-image: linear-gradient(to right, red, blue);" -->
<div class="bg-[linear-gradient(to_right,_red,_blue)]"></div>
```

We can register arbitrary `backgroundImage` utility classes with:

- a `url`: a `url()` function that specifies a resource
- an `image`-like: this could be a `linear-gradient()`, `radial-gradient()`, etc.

So, to ensure we only register the right arbitrary `backgroundImage` utility classes, we can use the `type` option in the `matchUtilities()` function to accept the `url` and `image` types:

```jsx
matchUtilities(
  {
    bg: (val) => {
      return {
        "background-image": val,
      };
    },
  },
  {
    values: theme("backgroundImage"),
    type: ["image", "url"],
  }
);
```

With this, we have every `bg-gradient-to-*` utility class under our control, and can move on to adding support for interpolation methods!

### Adding support for interpolation methods

There are several types of interpolation methods that we could use; we can control this with a simple CSS variable. We don’t want to initialize this variable to some default, since we know there isn’t a default color interpolation method that browsers use (it should but `oklab`, but isn’t). The CSS `var()` has special syntax that allows us to define a fallback:

```css
.gradient {
  background-image: linear-gradient(
    to right var(--interp-method, in oklab),
    red,
    blue
  );
}
```

We can leave this empty, which will give us an empty fallback:

```css
.gradient {
  background-image: linear-gradient(to right var(--interp-method), red, blue);
}
```

We’ll say that `--tw-color-interpolation-method` sets the color interpolation method for our gradient. We can pretty easily update the syntax in our original `matchUtilities()` function:

```jsx
matchUtilities(
  {
    "bg-gradient-to": (val) => {
      return {
        "background-image": `linear-gradient(${val} var(--tw-color-interpolation-method, ), var(--tw-gradient-stops,))`,
      };
    },
  },
  {
    values: theme("gradientDirection"),
  }
);
```

By default, `-tw-color-interpolation-method` isn't defined, and our gradient interpolates in the default color space. To change this variable, we can override it in a separate set of utility classes. Since these are static and we _don’t_ want the user to be able to define their own arbitrary values, we can use the `addUtilities()` plugin function, which allows us to define static utilities we don’t want the user to be able to change.

Earlier, I briefly mentioned rectangular and polar color spaces. We’ll store these color space names in their own arrays, and then iterate through these arrays to generate a set of static utility classes:

```jsx
const rectangularSpaces = ["srgb", "srgb-linear", "lab", "oklab", "xyz"];
const polarSpaces = ["hsl", "hwb", "lch", "oklch"];

for (const space of [...rectangularSpaces, ...polarSpaces]) {
  addUtilities({
    [`.bg-interpolate-${space}`]: {
      "--tw-color-interpolation-method": `in ${space}`,
    },
  });
}
```

Any given polar color space can _optionally_ have one of four interpolation methods, so we need to account for this. We can tap into the existing `/` syntax in Tailwind, which seems to be used most frequently in “optional specifiers.” For example, we can set an element’s color using `color-blue-500`, but we can optionally specify an opacity for that color using the `/` syntax: `color-blue-500/50`. Another place this is used is when we need to optionally specify a unique identifier for a `group` or `peer` class: `group/card`. I think this syntax is a great way to support relevant and optional information in a concise format, so we’ll shamelessly plunder it.

Like, there really isn’t a need for an entire set of utility classes just for hue interpolation methods:

```html
<div class="bg-gradient-to-330 bg-interpolate-oklch bg-interpolate-hue-longer">
</div>
```

When instead, we can just specify the method right in our color space utility class with the `/` modifier`:`

```html
<div class="bg-gradient-to-225 bg-interpolate-hwb/longer"></div>
```

We’ll loop through just the cylindrical color spaces, and add some more static utility classes to cover the optional `<hue-interpolation-method>` data type:

<!-- note: we need to escape the backslashes which is why there are four in the below codeblock lol -->

```jsx
for (const space of polarSpaces) {
  const hueInterpMethod = ["longer", "shorter", "increasing", "decreasing"];
  for (const interpMethod of hueInterpMethod) {
    addUtilities({
      // we escape the "/" with "\\\\"
      [`.bg-interpolate-${space}\\\\/${interpMethod}`]: {
        "--tw-color-interpolation-method": `${space} ${interpMethod} hue`,
      },
    });
  }
}
```

With this, we have a really nice expanded set of classes we can use to better control our gradients :D

```html
<!-- rectangular color space interpolation -->
<div class="bg-gradient-to-r bg-interpolate-oklab from-red-500 to-blue-500">
</div>

<!-- cylindrical color space interpolation -->
<!-- this is visually equivalent to using the "shorter hue" <hue-interpolation-method> -->
<div class="bg-gradient-to-[50deg] bg-interpolate-hwb from-red-500 to-blue-500">
</div>

<!-- cylindrical color space interpolation: with <hue-interpolation-method> -->
<div class="bg-gradient-to-[27deg] bg-interpolate-hwb/longer from-red-500 to-blue-500">
</div>
<div class="bg-gradient-to-12 bg-interpolate-hwb/shorter from-red-500 to-blue-500">
</div>
<div class="bg-gradient-to-bl bg-interpolate-oklch/increasing from-red-500 to-blue-500">
</div>
<div class="bg-gradient-to-r bg-interpolate-oklch/decreasing from-red-500 to-blue-500">
</div>
```

### Adding browser fallbacks

However, there’s one more problem we need to take care of! Some browsers, like Firefox, currently don’t support this new syntax. Gradients with a specified color interpolation method wouldn’t even render on browsers that don’t support the new syntax.

We _could_ test to see if the browser is Firefox with the following `@supports()` CSS function:

```css
@supports (-moz-appearance: none) {
  /* ... */
}
```

But this isn’t very graceful, since we would need to manually remove this once Firefox gets support for the new syntax. Any other browser without support for this syntax would also have to be included with its own `@supports` rule. Instead of trying to capture every since browser that doesn’t support the new gradient syntax, we can instead test for browsers that _do_. In other words, we don’t bother implementing a graceful fallback; rather, we implement **progressive enhancement**.

With this, we can change our theoretical `@supports` CSS function. If we want to see if the browser supports interpolating in `srgb`, we can just check if the corresponding `background-image` declaration is valid:

```css
/* .bg-interpolate-srgb */
@supports (background-image: linear-gradient(in srgb, red, red)) {
  .bg-interpolate-srgb {
    --tw-color-interpolation-method: "in srgb";
  }
}
```

Tailwind’s utility registration functions are really flexible in how we register new class rules since a CSS rule is represented as an object, and we can represent an at-rule and its relevant CSS rules as a nested object. We can modify both of our `addUtilities()` functions to support this progressive enhancement:

```jsx
const rectangularSpaces = ["srgb", "srgb-linear", "lab", "oklab", "xyz"];
const polarSpaces = ["hsl", "hwb", "lch", "oklch"];

// Add classes for default rectangular and cylindrical spaces
for (const space of [...rectangularSpaces, ...polarSpaces]) {
  addUtilities({
    [`@supports (background-image: linear-gradient(in ${space}, red, red))`]: {
      [`.bg-interpolate-${space}`]: {
        "--tw-color-interpolation-method": `in ${space}`,
      },
    },
  });
}

// add optional variant classes for polar color space interpolation methods that
// *also* specify a hue interpolation method
for (const space of polarSpaces) {
  const hueInterpMethod = ["longer", "shorter", "increasing", "decreasing"];

  for (const interpMethod of hueInterpMethod) {
    addUtilities({
      [`@supports (background-image: linear-gradient(in ${space}, red, red))`]:
        {
          [`.bg-interpolate-${space}\\/${interpMethod}`]: {
            "--tw-color-interpolation-method": `in ${space} ${interpMethod} hue`,
          },
        },
    });
  }
}
```

And, hey! That’s actually all there is to it! With that, we have a really robust set of utility classes that allow us to specify the color interpolation method for our gradients. The best part is that these progressively enhance, so browsers that don’t currently support the new gradient syntax will automatically support these utility classes when they gain support—no library update required!

*(NOTE: the SSH version of this post can't properly display this utility class list... yet.)*

<!-- <div class="max-h-[80vh] overflow-auto border border-zinc-200 dark:border-zinc-800 my-4 rounded-md">
<table style="table-layout: fixed;">
  <caption class="py-3 italic bg-zinc-100 dark:bg-zinc-900">Implemented color interpolation method utility classes and generated CSS properties</caption>
  <colgroup>
      <col span="1" class="w-[30%]">
      <col span="1" class="w-[70%] overflow-x-scroll">
  </colgroup>
  <thead>
    <tr>
      <th>Tailwind class</th>
      <th>Generated declarations</th>
    </tr>
  </thead>

  <tbody>
    <tr>
      <td>

`bg-gradient-to-*`

      </td>
      <td>

`background-image: linear-gradient(<dir> in var(--tw-color-interpolation-method, ), var(--tw-gradient-stops,));`

    </td>

  </tr>

  <tr>
    <th colspan="2">Rectangular color interpolation methods</th>
  </tr>

  <tr>
    <td>

`bg-interpolate-srgb`

    </td>
    <td>

```css
@supports (background: linear-gradient(in srgb, red, red)) {
  .bg-interpolate-srgb {
    --tw-color-interpolation-method: srgb;
  }
}
```

    </td>

  </tr>

  <tr>
    <td>

`bg-interpolate-srgb-linear`

    </td>
    <td>

```css
@supports (background: linear-gradient(in srgb-linear, red, red)) {
  .bg-interpolate-srgb-linear {
    --tw-color-interpolation-method: srgb-linear;
  }
}
```

    </td>

  </tr>

  <tr>
    <td>

`bg-interpolate-lab`

    </td>
    <td>

```css
@supports (background: linear-gradient(in lab, red, red)) {
  .bg-interpolate-lab {
    --tw-color-interpolation-method: lab;
  }
}
```

    </td>

  </tr>

  <tr>
    <td>

`bg-interpolate-oklab`

    </td>
    <td>

```css
@supports (background: linear-gradient(in oklab, red, red)) {
  .bg-interpolate-oklab {
    --tw-color-interpolation-method: oklab;
  }
}
```

    </td>

  </tr>

  <tr>
    <td>

`bg-interpolate-xyz`

    </td>
    <td>

```css
@supports (background: linear-gradient(in xyz, red, red)) {
  .bg-interpolate-xyz {
    --tw-color-interpolation-method: xyz;
  }
}
```

    </td>

  </tr>

  <tr>
    <th colspan="2">Polar color interpolation methods</th>
  </tr>

  <tr>
    <td>

`bg-interpolate-hsl`

`bg-interpolate-hsl/shorter`

`bg-interpolate-hsl/longer`

`bg-interpolate-hsl/increasing`

`bg-interpolate-hsl/decreasing`

    </td>
    <td>

```css
@supports (background: linear-gradient(in hsl, red, red)) {
  .bg-interpolate-hsl {
    --tw-color-interpolation-method: hsl;
  }
  .bg-interpolate-hsl\\\\/shorter {
    --tw-color-interpolation-method: hsl shorter hue;
  }
  .bg-interpolate-hsl\\\\/longer {
    --tw-color-interpolation-method: hsl longer hue;
  }
  .bg-interpolate-hsl\\\\/increasing {
    --tw-color-interpolation-method: hsl increasing hue;
  }
  .bg-interpolate-hsl\\\\/decreasing {
    --tw-color-interpolation-method: hsl decreasing hue;
  }
}
```

    </td>

  </tr>

  <tr>
    <td>

`bg-interpolate-hwb`

`bg-interpolate-hwb/shorter`

`bg-interpolate-hwb/longer`

`bg-interpolate-hwb/increasing`

`bg-interpolate-hwb/decreasing`

    </td>
    <td>

```css
@supports (background: linear-gradient(in hwb, red, red)) {
  .bg-interpolate-hwb {
    --tw-color-interpolation-method: hwb;
  }
  .bg-interpolate-hwb\\\\/shorter {
    --tw-color-interpolation-method: hwb shorter hue;
  }
  .bg-interpolate-hwb\\\\/longer {
    --tw-color-interpolation-method: hwb longer hue;
  }
  .bg-interpolate-hwb\\\\/increasing {
    --tw-color-interpolation-method: hwb increasing hue;
  }
  .bg-interpolate-hwb\\\\/decreasing {
    --tw-color-interpolation-method: hwb decreasing hue;
  }
}
```

    </td>

  </tr>

  <tr>
    <td>

`bg-interpolate-lch`

`bg-interpolate-lch/shorter`

`bg-interpolate-lch/longer`

`bg-interpolate-lch/increasing`

`bg-interpolate-lch/decreasing`

    </td>
    <td>

```css
@supports (background: linear-gradient(in lch, red, red)) {
  .bg-interpolate-lch {
    --tw-color-interpolation-method: lch;
  }
  .bg-interpolate-lch\\\\/shorter {
    --tw-color-interpolation-method: lch shorter hue;
  }
  .bg-interpolate-lch\\\\/longer {
    --tw-color-interpolation-method: lch longer hue;
  }
  .bg-interpolate-lch\\\\/increasing {
    --tw-color-interpolation-method: lch increasing hue;
  }
  .bg-interpolate-lch\\\\/decreasing {
    --tw-color-interpolation-method: lch decreasing hue;
  }
}
```

    </td>

  </tr>

  <tr>
    <td>

`bg-interpolate-oklch`

`bg-interpolate-oklch/shorter`

`bg-interpolate-oklch/longer`

`bg-interpolate-oklch/increasing`

`bg-interpolate-oklch/decreasing`

    </td>
    <td>

```css
@supports (background: linear-gradient(in oklch, red, red)) {
  .bg-interpolate-oklch {
    --tw-color-interpolation-method: oklch;
  }
  .bg-interpolate-oklch\\\\/shorter {
    --tw-color-interpolation-method: oklch shorter hue;
  }
  .bg-interpolate-oklch\\\\/longer {
    --tw-color-interpolation-method: oklch longer hue;
  }
  .bg-interpolate-oklch\\\\/increasing {
    --tw-color-interpolation-method: oklch increasing hue;
  }
  .bg-interpolate-oklch\\\\/decreasing {
    --tw-color-interpolation-method: oklch decreasing hue;
  }
}
```

    </td>

  </tr>
  </tbody>
</table>
</div> -->

## Supporting other gradient functions

With our improvements, we’ve implemented essentially every feature available to us in a linear-gradient() function. These improvements give us granular control when it comes to defining linear gradients… but what about other gradients? If we want to support a conic gradient, for example, we need to either use square bracket notation for an arbitrary property…

```html
<div class="[background-image:conic-gradient(in_oklab,theme(colors.red.500),theme(colors.blue.500))]">
</div>
```

…or, we need to add a custom utility:

```css
@layer utilities {
  .conic-gradient {
    background-image: conic-gradient(
      in oklab,
      theme(colors.red.500),
      theme(colors.blue.500)
    );
  }
}
```

Besides this being cumbersome, we also end up losing that really nice progressive enhancement we originally got. There is no graceful fallback for the above CSS rule, so it’ll end up failing on browsers without `<color-interpolation-method>` support.

Beyond `linear-gradient()`, there are two other gradient functions that we’ll aim to support. The first is `radial-gradient()`, which specifies a gradient that starts at an origin and “radiates” outwards. We can also use a `conic-gradient()`, which specifies a gradient that interpolates _around_ the circle—kind of like a color wheel.

### How similar is gradient syntax?

The syntax between all three gradient functions is similar in some ways, and different in others.

```html
<linear-gradient-syntax> = [ [ <angle> | to <side-or-corner> ] || <color-interpolation-method> ]?, <color-stop-list>

<radial-gradient-syntax> = [ [ [ <radial-shape> || <radial-size> ]? [ at <position> ]? ] || <color-interpolation-method>]?, <color-stop-list>

<conic-gradient-syntax> = [ [ [ from <angle> ]? [ at <position> ]? ] || <color-interpolation-method>]?, <angular-color-stop-list>
```

Linear and radial gradients use the same `<color-stop-list>` data type, while conic gradients use a special `<angular-color-stop-list>`. This seems like it might be a problem since it would be a huge pain to have to redefine _every_ color stop utility for conic gradients. However, by expanding these data types, we can see just how similar they really are:

```html
<!-- linear color stop list: used in linear and radial gradients -->
<color-stop-list> = <color> <length-percentage>{1,2}, [<length-percentage>?, <color> <length-percentage>{1,2} ]#

<!-- angular color stop list: used in conic gradients -->
<angular-color-stop-list> = <color> <angle-percentage>{1,2}, [<angle-percentage>?, <color> <angle-percentage>{1,2} ]#
```

The only difference between `<color-stop-list>` and `<angular-color-stop-list>` is in the data type we use to determine where to place a color stop. `<color-stop-list>` uses the `<length-percentage>` data type, and `<angular-color-stop-list>` uses the `<angle-percentage>` data type:

```html
<length-percentage> = <length> | <percentage>

<angle-percentage> = <angle> | <percentage>
```

Earlier, we found that we can’t easily target arbitrary angle values like to-[45deg], since Tailwind doesn’t expose any angle type to us in the matchUtilities() type option. We can also dive into the Tailwind source code and see that generating color stops is a pretty complex process…and not one we can really override. With that said, we’ll stick to support <percentage> by default. The good thing with this is that the color stop syntax is—as far as we’re concerned—entirely identical between all three gradient functions:

```html
<general-color-stop-list> = <color> <percentage>{1,2}, [<percentage>?, <color> <percentage>{1,2} ]#
```

In other words, this isn’t something we need to worry about! We can use the same `--tw-gradient-stops` variable to specify our color stops.

Before we dive into registering a bunch of classes to determine the position of these gradients, let’s first register two static utilities to specify whether we’re using a radial or conic gradient. These classes mirror `bg-gradient-to-*`, since they specify the gradient function itself:

```jsx
addUtilities({
  ".bg-gradient-radial": {
    "background-image":
      "radial-gradient(var(--tw-color-interpolation-method, ), var(--tw-gradient-stops))",
  },

  ".bg-gradient-conic": {
    "background-image":
      "conic-gradient(var(--tw-color-interpolation-method, ), var(--tw-gradient-stops))",
  },
});
```

### Implementing basic origin positioning

All three gradient functions have different ways of specifying how they are rendered with respect to the element they are in. Linear gradients can specify a direction they “point” in. Radial gradients can specify an “origin,” as well as an overall shape. The syntax does differ quite a bit between these three functions:

- Linear gradients: `<angle> | to <side-or-corner>`
- Radial gradients: `[ at <position> ]? [ <radial-shape> || <radial-size> ]?`
- Conic gradients: `[ at <position> ]? [ from <angle> ]?`

Radial and conic gradients both have _two_ optional positioning components. The first one that we’ll work out is the `at <position>` component since its syntax is the same for radial and conic gradients. Later, we’ll implement the other components of this positioning syntax. The syntax for the `<position>` data type can get pretty complex:

The syntax for the `<position>` data type can get pretty complex:

```html
[ at <position> ]?

<position> = [
  [ left | center | right | top | bottom | <length-percentage> ] |
  [ left | center | right ] && [ top | center | bottom ] |
  [ left | center | right |<length-percentage> ] [ top | center | bottom | <length-percentage> ] |
  [ [ left | right ] <length-percentage> ] && [ [ top | bottom ] <length-percentage> ]
]
```

This data type essentially breaks down into four main cases:

1.  **no positioning**: in this case, we don’t provide any `at <position>` syntax, and `at center center` is implied:

    ```css
    /* no positioning: */
    background-image: radial-gradient(red, blue);
    /* is equivalent to */
    background-image: radial-gradient(at center center, red, blue);
    ```

2.  **a single keyword _or_ `<length-percentage>`**: in this case, we provide either a “keyword” (`left`, `right`, `top`, etc.), or a `<length-percentage>` (40%, 20px, etc.). The resulting gradient uses our supplied value to horizontally position itself—its vertical position is still `center`:

    ```css
    /* a single keyword ([left | center | right | top | bottom]): */
    background-image: conic-gradient(at left, red, blue);
    /* is equivalent to: */
    background-image: conic-gradient(at left center, red, blue);

    /* a single length *or* percentage (<length-percentage>):  */
    background-image: radial-gradient(at 40%, red, blue);
    /* is equivalent to: */
    background-image: radial-gradient(at 40% center, red, blue);
    ```

3.  **two keywords or `<length-percentage>` values**: we define both a horizontal and vertical origin:

    ```css
    /* two keywords or <length-percentage> values: */
    background-image: conic-gradient(at left 40%, red, blue);
    background-image: radial-gradient(at 30px top, red, blue);
    background-image: radial-gradient(at right bottom, red, blue);
    background-image: radial-gradient(at 2px 4px, red, blue);
    ```

4.  **four components**: here, we define horizontal and vertical “transform origins” with a keyword, followed by an offset with a `<length-percentage>`. `top 40px`, for example, will position the vertical origin of the gradient 40px from the top of the element; `bottom 40px`, on the other hand, positions the vertical origin of the gradient 40px from the bottom of the element:

    ```css
    /* four (!) components: keyword and <length-percentage> for both axes */
    background-image: conic-gradient(at left 40% top 65%, red, blue);
    background-image: radial-gradient(at right 20px bottom 10%, red, blue);
    ```

In the first and second cases, the default value for any missing component is `center`. Because of this, the first and second cases both boil down to our two-component syntax, and we don’t need to explicitly handle them :D

For our sanity, we will leave the four-component syntax unsupported. We want to strike a balance between capability and complexity, and registering a bunch of utility classes to handle this relatively rare case is a bit unnecessary. If someone needs the four-component syntax they can always use an arbitrary value.

We can represent the horizontal and vertical positions of a radial or conic gradient as CSS variables, and default them to `center`. We’ll add these variables to the utility classes we just registered, and then override them elsewhere:

```jsx
addUtilities({
  ".bg-gradient-radial": {
    "--tw-gradient-x-position": "center",
    "--tw-gradient-y-position": "center",
    "background-image":
      "radial-gradient(at var(--tw-gradient-x-position) var(--tw-gradient-y-position) var(--tw-color-interpolation-method, ), var(--tw-gradient-stops))",
  },

  ".bg-gradient-conic": {
    "--tw-gradient-x-position": "center",
    "--tw-gradient-y-position": "center",
    "background-image":
      "conic-gradient(at var(--tw-gradient-x-position) var(--tw-gradient-y-position) var(--tw-color-interpolation-method, ), var(--tw-gradient-stops))",
  },
});
```

We’ll then define dynamic utilities that can override `--tw-gradient-x-position`, `--tw-gradient-y-position`, or both at the same time. If we’re setting both X and Y positions at the same time, then we can define 8 keyword cases: `top center`, `top right`, `center right`, `bottom right`, etc. If we’re setting only one position variable, then we can set either `left` and `right` or `top` and `bottom`, depending on the direction. Given this, we’ll take a slightly modified approach to registering these classes.

We’ll fill our a custom theme property called `gradientPosition`, which can accept a set of percentages (the percentage values mirror what many Tailwind utilities—like `width`—typically accept):

```jsx
export default {
  theme: {
    // ...
    gradientPosition: ({ theme }) => ({
      ...theme("percentage"),
    }),
    percentage: {
      "1/2": "50%",
      "1/3": "33.333333%",
      "2/3": "66.666667%",
      "1/4": "25%",
      "2/4": "50%",
      "3/4": "75%",
      "1/5": "20%",
      "2/5": "40%",
      "3/5": "60%",
      "4/5": "80%",
      "1/6": "16.666667%",
      "2/6": "33.333333%",
      "3/6": "50%",
      "4/6": "66.666667%",
      "5/6": "83.333333%",
      "1/12": "8.333333%",
      "2/12": "16.666667%",
      "3/12": "25%",
      "4/12": "33.333333%",
      "5/12": "41.666667%",
      "6/12": "50%",
      "7/12": "58.333333%",
      "8/12": "66.666667%",
      "9/12": "75%",
      "10/12": "83.333333%",
      "11/12": "91.66667%",
      full: "100%",
    },
    // ...
  },
  // ..
  plugins: [
    /* ... */
  ],
};
```

Once we have this, we’ll register the keyword-based utility classes as _static_ classes, and the percentage-based utility classes as dynamic classes (i.e. using the `gradientPosition` class):

```jsx
// CASE 1: two-component syntax
const corners = [
  ["t", "top center"],
  ["tr", "top right"],
  ["r", "center right"],
  ["br", "bottom right"],
  ["b", "bottom center"],
  ["bl", "bottom left"],
  ["l", "center left"],
  ["tl", "top left"],
];

for (const [shorthand, value] of corners) {
  addUtilities({
    [`.bg-gradient-pos-${shorthand}`]: {
      "--tw-gradient-x-position": value.split(" ")[0],
      "--tw-gradient-y-position": value.split(" ")[1],
    },
  });
}

matchUtilities(
  {
    "bg-gradient-pos": (val) => {
      const splitIdx = val.indexOf(" ");

      let x = val;
      let y = val;
      if (splitIdx > -1) {
        x = val.substring(0, splitIdx);
        y = val.substring(splitIdx + 1);
      }

      return {
        "--tw-gradient-x-position": x,
        "--tw-gradient-y-position": y,
      };
    },
  },
  {
    type: "any",
    values: theme("gradientPosition"),
  }
);

// CASE 1: one-component syntax: X directions
const xEdges = [
  ["l", "left"],
  ["r", "right"],
];

for (const [shorthand, value] of xEdges) {
  addUtilities({
    [`.bg-gradient-pos-x-${shorthand}`]: {
      "--tw-gradient-x-position": value,
    },
  });
}

matchUtilities(
  {
    "bg-gradient-pos-x": (val) => {
      return {
        "--tw-gradient-x-position": val,
      };
    },
  },
  {
    type: "any",
    values: theme("gradientPosition"),
  }
);

// CASE 3: one-component: syntax: Y directions

const yEdges = [
  ["t", "top"],
  ["b", "bottom"],
];

for (const [shorthand, value] of yEdges) {
  addUtilities({
    [`.bg-gradient-pos-y-${shorthand}`]: {
      "--tw-gradient-y-position": value,
    },
  });
}
```

It’s a lot of code, but nothing too bad. One thing to note is the `matchUtilities()` function for the dual-direction `bg-gradient-pos-*` classes. If the class is something like `bg-gradient-pos-[20px]`, then we apply `20px` to both variables:

```css
.bg-gradient-pos-[20px] {
  --tw-gradient-x-position: 20px,
  --tw-gradient-y-position: 20px,
}
```

If the class is instead something like `bg-gradient-pos-[20px_40px]`, then the plugin recognizes we’re sending two values (the `_` character is interpreted as a space in Tailwind):

```css
.bg-gradient-pos-[20px_40px] {
  --tw-gradient-x-position: 20px,
  --tw-gradient-y-position: 40px,
}
```

With that, we have an extensive set of utilities to let us position both radial and conic gradients pretty much anywhere.

| Tailwind class            | Generated declarations                                                             |
| ------------------------- | ---------------------------------------------------------------------------------- |
| `bg-gradient-pos-t`       | `--tw-gradient-x-position: center;`<br>`--tw-gradient-y-position: top;`            |
| `bg-gradient-pos-tr`      | `--tw-gradient-x-position: right;`<br>`--tw-gradient-y-position: top;`             |
| `bg-gradient-pos-r`       | `--tw-gradient-x-position: right;`<br>`--tw-gradient-y-position: center;`          |
| `bg-gradient-pos-br`      | `--tw-gradient-x-position: right;`<br>`--tw-gradient-y-position: bottom;`          |
| `bg-gradient-pos-b`       | `--tw-gradient-x-position: center;`<br>`--tw-gradient-y-position: bottom;`         |
| `bg-gradient-pos-bl`      | `--tw-gradient-x-position: left;`<br>`--tw-gradient-y-position: bottom;`           |
| `bg-gradient-pos-l`       | `--tw-gradient-x-position: left;`<br>`--tw-gradient-y-position: center;`           |
| `bg-gradient-pos-tl`      | `--tw-gradient-x-position: left;`<br>`--tw-gradient-y-position: top;`              |
| `bg-gradient-pos-1/2`     | `--tw-gradient-x-position: 50%;`<br>`--tw-gradient-y-position: 50%;`               |
| `bg-gradient-pos-1/3`     | `--tw-gradient-x-position: 33.333333%;`<br>`--tw-gradient-y-position: 33.333333%;` |
| `bg-gradient-pos-2/3`     | `--tw-gradient-x-position: 66.666667%;`<br>`--tw-gradient-y-position: 66.666667%;` |
| `bg-gradient-pos-1/4`     | `--tw-gradient-x-position: 25%;`<br>`--tw-gradient-y-position: 25%;`               |
| `bg-gradient-pos-2/4`     | `--tw-gradient-x-position: 50%;`<br>`--tw-gradient-y-position: 50%;`               |
| `bg-gradient-pos-3/4`     | `--tw-gradient-x-position: 75%;`<br>`--tw-gradient-y-position: 75%;`               |
| `bg-gradient-pos-1/5`     | `--tw-gradient-x-position: 20%;`<br>`--tw-gradient-y-position: 20%;`               |
| `bg-gradient-pos-2/5`     | `--tw-gradient-x-position: 40%;`<br>`--tw-gradient-y-position: 40%;`               |
| `bg-gradient-pos-3/5`     | `--tw-gradient-x-position: 60%;`<br>`--tw-gradient-y-position: 60%;`               |
| `bg-gradient-pos-4/5`     | `--tw-gradient-x-position: 80%;`<br>`--tw-gradient-y-position: 80%;`               |
| `bg-gradient-pos-1/6`     | `--tw-gradient-x-position: 16.666667%;`<br>`--tw-gradient-y-position: 16.666667%;` |
| `bg-gradient-pos-2/6`     | `--tw-gradient-x-position: 33.333333%;`<br>`--tw-gradient-y-position: 33.333333%;` |
| `bg-gradient-pos-3/6`     | `--tw-gradient-x-position: 50%;`<br>`--tw-gradient-y-position: 50%;`               |
| `bg-gradient-pos-4/6`     | `--tw-gradient-x-position: 66.666667%;`<br>`--tw-gradient-y-position: 66.666667%;` |
| `bg-gradient-pos-5/6`     | `--tw-gradient-x-position: 83.333333%;`<br>`--tw-gradient-y-position: 83.333333%;` |
| `bg-gradient-pos-1/12`    | `--tw-gradient-x-position: 8.333333%;`<br>`--tw-gradient-y-position: 8.333333%;`   |
| `bg-gradient-pos-2/12`    | `--tw-gradient-x-position: 16.666667%;`<br>`--tw-gradient-y-position: 16.666667%;` |
| `bg-gradient-pos-3/12`    | `--tw-gradient-x-position: 25%;`<br>`--tw-gradient-y-position: 25%;`               |
| `bg-gradient-pos-4/12`    | `--tw-gradient-x-position: 33.333333%;`<br>`--tw-gradient-y-position: 33.333333%;` |
| `bg-gradient-pos-5/12`    | `--tw-gradient-x-position: 41.666667%;`<br>`--tw-gradient-y-position: 41.666667%;` |
| `bg-gradient-pos-6/12`    | `--tw-gradient-x-position: 50%;`<br>`--tw-gradient-y-position: 50%;`               |
| `bg-gradient-pos-7/12`    | `--tw-gradient-x-position: 58.333333%;`<br>`--tw-gradient-y-position: 58.333333%;` |
| `bg-gradient-pos-8/12`    | `--tw-gradient-x-position: 66.666667%;`<br>`--tw-gradient-y-position: 66.666667%;` |
| `bg-gradient-pos-9/12`    | `--tw-gradient-x-position: 75%;`<br>`--tw-gradient-y-position: 75%;`               |
| `bg-gradient-pos-10/12`   | `--tw-gradient-x-position: 83.333333%;`<br>`--tw-gradient-y-position: 83.333333%;` |
| `bg-gradient-pos-11/12`   | `--tw-gradient-x-position: 91.66667%;`<br>`--tw-gradient-y-position: 91.66667%;`   |
| `bg-gradient-pos-full`    | `--tw-gradient-x-position: 100%;`<br>`--tw-gradient-y-position: 100%;`             |
| `bg-gradient-pos-x-l`     | `--tw-gradient-x-position: left;`                                                  |
| `bg-gradient-pos-x-r`     | `--tw-gradient-x-position: right;`                                                 |
| `bg-gradient-pos-x-1/2`   | `--tw-gradient-x-position: 50%;`                                                   |
| `bg-gradient-pos-x-1/3`   | `--tw-gradient-x-position: 33.333333%;`                                            |
| `bg-gradient-pos-x-2/3`   | `--tw-gradient-x-position: 66.666667%;`                                            |
| `bg-gradient-pos-x-1/4`   | `--tw-gradient-x-position: 25%;`                                                   |
| `bg-gradient-pos-x-2/4`   | `--tw-gradient-x-position: 50%;`                                                   |
| `bg-gradient-pos-x-3/4`   | `--tw-gradient-x-position: 75%;`                                                   |
| `bg-gradient-pos-x-1/5`   | `--tw-gradient-x-position: 20%;`                                                   |
| `bg-gradient-pos-x-2/5`   | `--tw-gradient-x-position: 40%;`                                                   |
| `bg-gradient-pos-x-3/5`   | `--tw-gradient-x-position: 60%;`                                                   |
| `bg-gradient-pos-x-4/5`   | `--tw-gradient-x-position: 80%;`                                                   |
| `bg-gradient-pos-x-1/6`   | `--tw-gradient-x-position: 16.666667%;`                                            |
| `bg-gradient-pos-x-2/6`   | `--tw-gradient-x-position: 33.333333%;`                                            |
| `bg-gradient-pos-x-3/6`   | `--tw-gradient-x-position: 50%;`                                                   |
| `bg-gradient-pos-x-4/6`   | `--tw-gradient-x-position: 66.666667%;`                                            |
| `bg-gradient-pos-x-5/6`   | `--tw-gradient-x-position: 83.333333%;`                                            |
| `bg-gradient-pos-x-1/12`  | `--tw-gradient-x-position: 8.333333%;`                                             |
| `bg-gradient-pos-x-2/12`  | `--tw-gradient-x-position: 16.666667%;`                                            |
| `bg-gradient-pos-x-3/12`  | `--tw-gradient-x-position: 25%;`                                                   |
| `bg-gradient-pos-x-4/12`  | `--tw-gradient-x-position: 33.333333%;`                                            |
| `bg-gradient-pos-x-5/12`  | `--tw-gradient-x-position: 41.666667%;`                                            |
| `bg-gradient-pos-x-6/12`  | `--tw-gradient-x-position: 50%;`                                                   |
| `bg-gradient-pos-x-7/12`  | `--tw-gradient-x-position: 58.333333%;`                                            |
| `bg-gradient-pos-x-8/12`  | `--tw-gradient-x-position: 66.666667%;`                                            |
| `bg-gradient-pos-x-9/12`  | `--tw-gradient-x-position: 75%;`                                                   |
| `bg-gradient-pos-x-10/12` | `--tw-gradient-x-position: 83.333333%;`                                            |
| `bg-gradient-pos-x-11/12` | `--tw-gradient-x-position: 91.66667%;`                                             |
| `bg-gradient-pos-x-full`  | `--tw-gradient-x-position: 100%;`                                                  |
| `bg-gradient-pos-y-t`     | `--tw-gradient-y-position: top;`                                                   |
| `bg-gradient-pos-y-b`     | `--tw-gradient-y-position: bottom;`                                                |
| `bg-gradient-pos-y-1/2`   | `--tw-gradient-y-position: 50%;`                                                   |
| `bg-gradient-pos-y-1/3`   | `--tw-gradient-y-position: 33.333333%;`                                            |
| `bg-gradient-pos-y-2/3`   | `--tw-gradient-y-position: 66.666667%;`                                            |
| `bg-gradient-pos-y-1/4`   | `--tw-gradient-y-position: 25%;`                                                   |
| `bg-gradient-pos-y-2/4`   | `--tw-gradient-y-position: 50%;`                                                   |
| `bg-gradient-pos-y-3/4`   | `--tw-gradient-y-position: 75%;`                                                   |
| `bg-gradient-pos-y-1/5`   | `--tw-gradient-y-position: 20%;`                                                   |
| `bg-gradient-pos-y-2/5`   | `--tw-gradient-y-position: 40%;`                                                   |
| `bg-gradient-pos-y-3/5`   | `--tw-gradient-y-position: 60%;`                                                   |
| `bg-gradient-pos-y-4/5`   | `--tw-gradient-y-position: 80%;`                                                   |
| `bg-gradient-pos-y-1/6`   | `--tw-gradient-y-position: 16.666667%;`                                            |
| `bg-gradient-pos-y-2/6`   | `--tw-gradient-y-position: 33.333333%;`                                            |
| `bg-gradient-pos-y-3/6`   | `--tw-gradient-y-position: 50%;`                                                   |
| `bg-gradient-pos-y-4/6`   | `--tw-gradient-y-position: 66.666667%;`                                            |
| `bg-gradient-pos-y-5/6`   | `--tw-gradient-y-position: 83.333333%;`                                            |
| `bg-gradient-pos-y-1/12`  | `--tw-gradient-y-position: 8.333333%;`                                             |
| `bg-gradient-pos-y-2/12`  | `--tw-gradient-y-position: 16.666667%;`                                            |
| `bg-gradient-pos-y-3/12`  | `--tw-gradient-y-position: 25%;`                                                   |
| `bg-gradient-pos-y-4/12`  | `--tw-gradient-y-position: 33.333333%;`                                            |
| `bg-gradient-pos-y-5/12`  | `--tw-gradient-y-position: 41.666667%;`                                            |
| `bg-gradient-pos-y-6/12`  | `--tw-gradient-y-position: 50%;`                                                   |
| `bg-gradient-pos-y-7/12`  | `--tw-gradient-y-position: 58.333333%;`                                            |
| `bg-gradient-pos-y-8/12`  | `--tw-gradient-y-position: 66.666667%;`                                            |
| `bg-gradient-pos-y-9/12`  | `--tw-gradient-y-position: 75%;`                                                   |
| `bg-gradient-pos-y-10/12` | `--tw-gradient-y-position: 83.333333%;`                                            |
| `bg-gradient-pos-y-11/12` | `--tw-gradient-y-position: 91.66667%;`                                             |
| `bg-gradient-pos-y-full`  | `--tw-gradient-y-position: 100%;`                                                  |

### Implementing other gradient arguments: `radial-gradient()` ending shape and `conic-gradient()` rotation angle

We’re almost done! All we need to do now is handle the unique syntax components in the radial and conic gradient functions.

To review the positioning syntax:

- Radial gradients: `[ at <position> ]? [ <radial-shape> || <radial-size> ]?`
- Conic gradients: `[ at <position> ]? [ from <angle> ]?`

**I. Radial Gradients**

The `[ <radial-shape> || <radial-size> ]?` component in the `radial-gradient()` function controls the _ending shape_ of the gradient: which is the ellipse formed at the end of the radial gradient. By default, this is an ellipse whose dimensions are such that the ellipse touches each edge of the element.

`<radial-shape>` is either `circle` or `ellipse` (defaults to `ellipse`), and tells us whether the ending shape is a circle or ellipse.

`<radial-size>` determines the dimensions of the ending shape. This data type can take on several values, however, we’ll focus on the primary case where the ending shape is determined by one of four `<radial-extent>` keywords:

- `closest-side`: the ending shape’s dimensions are such that the shape meets the closest edge(s) from the gradient’s center.
- `farthest-side`: the ending shape’s dimensions are such that the shape meets the furthest edge(s) from the gradient’s center. _this is the default if no `<radial-size>` value is provided._
- `closest-corner`: the ending shape’s dimensions are such that the shape—scaled from the gradient’s center—meets the closest corner from the gradient’s center.
- `farthest-corner`: the ending shape’s dimensions are such that the shape—scaled from the gradient’s center—meets the furthest corner from the gradient’s center.

There are also two other data types we can set the `<radial-size>` to, which depend on whether `<radial-shape>` is a `circle` or `ellipse`:

- if `<radial-shape>` is `circle`: `<radial-size>` may be a single _absolute_ length, like `4rem` or `7px`
- if `<radial-shape>` is `ellipse`: `<radial-size>` may be 2 absolute lengths or relative percentages.

We’ll limit support for these to just arbitrary values. Let’s start with updating our `.bg-gradient-radial` utility class to handle these new components:

```jsx
addUtilities({
  ".bg-gradient-radial": {
    "--tw-gradient-x-position": "center",
    "--tw-gradient-y-position": "center",
    "--tw-radial-shape": "ellipse",
    "--tw-radial-size": "farthest-corner",
    "background-image":
      "radial-gradient(var(--tw-radial-shape) var(--tw-radial-size) at var(--tw-gradient-x-position) var(--tw-gradient-y-position) var(--tw-color-interpolation-method, ), var(--tw-gradient-stops))",
  },

  // ...
});
```

We’ll first handle the `<radial-shape>` since it can only be one of two options:

```jsx
addUtilities({
  "radial-grad-circle": {
    "--tw-radial-shape": "circle",
  },
  "radial-grad-ellipse": {
    "--tw-radial-shape": "ellipse",
  },
});
```

For `<radial-size>`, we’ll start by registering our default cases (the keywords described by `<radial-extent>`) to a new custom theme property `radialGradientSize`:

```jsx
export default {
  theme: {
    // ...
    radialGradientSize: {
      "closest-side": "closest-side",
      "farthest-side": "farthest-side",
      "closest-corner": "closest-corner",
      "farthest-corner": "farthest-corner",
    },
    // ...
  },
  // ...
};
```

Then, we’ll register these into a set of dynamic utilities. We also want to respect arbitrary values; however, if we specify the `type` option to restrict our types to lengths and percentages, then we can’t specify more than a single arbitrary value. In other words, a class like `gradient-extent-[5rem]` would work just fine, but `gradient-extent-[5rem_5rem]` wouldn’t. It seems like Tailwind parses the type of the entire passed-in string _before_ it splits the `_` delimiter—and `5rem_5rem` doesn’t fall into any type. Because of this, we’ll leave the `type` option out and lazily parse whatever arbitrary value the user provides:

```jsx
matchUtilities(
  {
    "radial-grad-extent": (val) => {
      return {
        "--tw-radial-size": val,
      };
    },
  },
  {
    values: theme("radialGradientSize"),
  }
);
```

This gives us a nice set of utility classes for controlling the size of a radial gradient.

| Tailwind class                  | Generated declarations             |
| ------------------------------- | ---------------------------------- |
| gradient-extent-closest-side    | --tw-radial-size: closest-side;    |
| gradient-extent-farthest-side   | --tw-radial-size: farthest-side;   |
| gradient-extent-closest-corner  | --tw-radial-size: closest-corner;  |
| gradient-extent-farthest-corner | --tw-radial-size: farthest-corner; |

**II. Conic Gradients**

The `[ from <angle> ]?` component allows us to specify the clockwise offset by which the overall gradient is rotated. There really isn’t that much more to it!

Just as we did with the `conic-gradient()` function, we’ll update our `.bg-gradient-conic` utility class:

```jsx
addUtilities({
  // ...
  ".bg-gradient-conic": {
    "--tw-gradient-x-position": "center",
    "--tw-gradient-y-position": "center",
    "--tw-conic-angle": "0deg",
    "background-image":
      "radial-gradient(from var(--tw-conic-angle) at var(--tw-gradient-x-position) var(--tw-gradient-y-position) var(--tw-color-interpolation-method, ), var(--tw-gradient-stops))",
  },
});
```

We can register a set of dynamic utilities purely based on the existing `rotate` theme property since we only need to deal with the `<angle>` data type:

```jsx
matchUtilities(
  {
    "conic-grad-angle": (val) => {
      return {
        "--tw-conic-angle": val,
      };
    },
  },
  {
    values: theme("rotate"),
  }
);
```

Phew! That was easy. We now have a set of utility classes to handle conic gradient offset angles:

| Tailwind class       | Generated declarations    |
| -------------------- | ------------------------- |
| conic-grad-angle-0   | --tw-conic-angle: 0deg;   |
| conic-grad-angle-1   | --tw-conic-angle: 1deg;   |
| conic-grad-angle-2   | --tw-conic-angle: 2deg;   |
| conic-grad-angle-3   | --tw-conic-angle: 3deg;   |
| conic-grad-angle-6   | --tw-conic-angle: 6deg;   |
| conic-grad-angle-12  | --tw-conic-angle: 12deg;  |
| conic-grad-angle-45  | --tw-conic-angle: 45deg;  |
| conic-grad-angle-90  | --tw-conic-angle: 90deg;  |
| conic-grad-angle-180 | --tw-conic-angle: 180deg; |

<!-- --- -->

## Summing it up

The source code for this project can be found [on my Github](https://github.com/maxmmyron/tailwind-extended-gradients), and you can also check out [a live demo](https://tailwind-extended-gradients.vercel.app/) that showcases all of the utility classes we’ve registered.

This blog is maybe a bit too long for what it really covers—6000 words (excluding code snippets) for a 300-line Tailwind plugin seems like a lot! However, through this journey, we learned quite a bit.

We dove into Tailwind’s fantastic and powerful plugin API. I occasionally found myself a bit stuck; for example: _how do we implement the `@supports` at-rule?_ I initially figured something like that would require some gross hack, but it didn’t! It’s baked right into how CSS components are defined, which is surprisingly beautiful. Of course, feel free to check out the [Plugin API documentation](https://tailwindcss.com/docs/plugins) for a more in-depth writeup of everything we used to implement this.

We learned a bit about color spaces! I mentioned them all already, but do check out Adam Argyle’s [guide to CSS color spaces](https://developer.chrome.com/docs/css-ui/high-definition-css-color-guide#color_interpolation), Eric Portis’ [color space explainer](https://ericportis.com/posts/2024/okay-color-spaces/), and Isaac Muse’s [interactive color space explorer](https://facelessuser.github.io/coloraide/demos/3d_models.html).

We dove into W3C syntax, and how to break it down into digestible components that we can work with. I never really found it super necessary to learn this syntax for everything I do on the web, however, it makes perusing through the W3C spec (and oddly entrancing hobby) a bit easier to do!

Of course, we explored how each gradient function differs from one another! Gradients allow us to make beautiful websites and apps, and this teeny Tailwind plugin gives us so much more control to take advantage of color interpolation, gradient positioning, and radial or conic gradients. What’s even better, we gain all of this while maintaining Tailwind’s utility-first fundamentals.
