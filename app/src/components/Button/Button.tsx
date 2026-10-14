import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { Link, type LinkProps } from 'react-router'
import { Icon } from '../Icon/Icon'
import type { IconName } from '../Icon/iconPaths'
import styles from './Button.module.css'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'text'
type Size = 'lg' | 'md' | 'sm'

type StyleProps = {
  /** primary：塗り／secondary：primary の枠／ghost：border の枠／danger：取り消し系の枠／text：文字だけ */
  variant?: Variant
  /** 高さ lg 52px（主要）／md 48px（副）／sm 44px（最小タップ領域） */
  size?: Size
  /** 親の幅いっぱいに広げる */
  fullWidth?: boolean
  /** 文字の前に置くアイコン */
  icon?: IconName
  children: ReactNode
}

function classNames({ variant = 'primary', size = 'lg', fullWidth }: StyleProps, extra?: string) {
  return [styles.button, styles[variant], styles[size], fullWidth && styles.fullWidth, extra]
    .filter(Boolean)
    .join(' ')
}

function Content({ icon, children }: Pick<StyleProps, 'icon' | 'children'>) {
  return (
    <>
      {icon && <Icon name={icon} size={20} />}
      {children}
    </>
  )
}

type ButtonProps = StyleProps & ButtonHTMLAttributes<HTMLButtonElement>

/** ボタン（DS-3）。押したときに画面内の処理をするもの。画面遷移だけなら ButtonLink を使う */
export function Button({
  variant,
  size,
  fullWidth,
  icon,
  children,
  className,
  type = 'button',
  ...rest
}: ButtonProps) {
  return (
    <button
      type={type}
      className={classNames({ variant, size, fullWidth, children }, className)}
      {...rest}
    >
      <Content icon={icon}>{children}</Content>
    </button>
  )
}

type ButtonLinkProps = StyleProps & Omit<LinkProps, 'children'>

/** ボタンの見た目のリンク（DS-3）。画面遷移に使う */
export function ButtonLink({
  variant,
  size,
  fullWidth,
  icon,
  children,
  className,
  ...rest
}: ButtonLinkProps) {
  return (
    <Link className={classNames({ variant, size, fullWidth, children }, className)} {...rest}>
      <Content icon={icon}>{children}</Content>
    </Link>
  )
}
